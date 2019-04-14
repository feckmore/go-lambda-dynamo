package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
)

func TestCreateSite(t *testing.T) {
	setup(t)

	var testCases = []struct {
		name     string
		in       *Site
		want     *Site
		wantCode int
	}{
		{"Nil Site", nil, nil, http.StatusBadRequest},
		{"Empty Site", &Site{}, nil, http.StatusBadRequest},
		{"Empty Name Only", &Site{Name: new(string)}, nil, http.StatusBadRequest},
		{"Description only", &Site{Description: aws.String("description")}, nil, http.StatusBadRequest},
		{"Name only", &Site{Name: aws.String("name")}, nil, http.StatusBadRequest},
		{"Path only", &Site{Path: "path"}, nil, http.StatusBadRequest},
		{"Name, Path", &Site{Name: aws.String("name"), Path: "path"}, &Site{Name: aws.String("name"), Path: "path"}, http.StatusOK},
		{"Name, Path, Status", &Site{Name: aws.String("name"), Path: "path", Status: Published}, &Site{Name: aws.String("name"), Path: "path", Status: Unpublished}, http.StatusOK},
	}

	for i, tc := range testCases {
		gotBody, gotCode, _ := invokeCreateSiteHandler(tc.in)

		t.Run(tc.name, func(t *testing.T) {
			if e := compareSites(tc.want, gotBody); e != nil {
				t.Errorf("%d: CreateSite Handler: got %+v; wanted %+v; error: %v", i, gotBody, tc.want, e)
			}

			if gotCode != tc.wantCode {
				t.Errorf("%d: CreateSite Handler: got code %v; wanted %v", i, gotCode, tc.wantCode)
			}
		})
	}
}

// ************************************
// internal testing helper functions... several could be moved to centralized location

func setup(t *testing.T) {
	region = "us-east-1"
	stage = "dev"              // CHANGE TO TEST
	table = "go-lambda-dynamo" // CHANGE TO TEST-SPECIFIC

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		t.Fatal("Error creating AWS session")
	} else {
		db = dynamodb.New(session)
	}
}

// invokeCreateSiteHandler marshals the input site to json, sends it to lambda in the request,
// then unmarshals & returns the result
func invokeCreateSiteHandler(in *Site) (*Site, int, error) {
	request := createSiteRequest(in)
	ctx := context.Background()
	response, err := Handler(ctx, request)
	if len(response.Body) == 0 {
		return nil, response.StatusCode, err
	}

	var out Site
	err = json.Unmarshal([]byte(response.Body), &out)
	if err != nil {
		log.Println("Error unmarshalling response body:", err)
	}

	return &out, response.StatusCode, err
}

// createSiteRequest takes a Site struct and returns a lambda request object containing the site as the body
func createSiteRequest(site *Site) Request {
	body, _ := json.Marshal(site)

	request := Request{
		Resource:   "/sites",
		Path:       "/sites",
		HTTPMethod: "Post",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}

	return request
}

// compareSites is used in place of reflect.DeepEqual for Sites because
// a) the ID is generated for new sites, and is not known, and
// b) the createdAt / updatedAt can't be known precisely
func compareSites(want, got *Site) error {
	var err error

	// Compare nil values. Exit if either is nil avoiding nil pointer dereference
	if want == nil || got == nil {
		if want != got {
			return errors.New("not both nil")
		}

		return nil
	}

	if e := compareIDs(want.ID, got.ID); e != nil {
		err = wrapError(err, e.Error())
	}

	// Compare status
	if want.Status != got.Status {
		fmt.Println(want.Status, got.Status)
		err = wrapError(err, "Status invalid")
	}

	// Compare fields
	if want.Name != nil && got.Name != nil && *want.Name != *got.Name {
		err = wrapError(err, "Names not equal")
	}
	if want.Description != nil && got.Description != nil && *want.Description != *got.Description {
		err = wrapError(err, "Names not equal")
	}

	currentTime := time.Now()
	// Compare dates: won't be identical if just created, so must be within short amout of time
	if got.CreatedAt.Sub(currentTime) > time.Second {
		err = wrapError(err, "CreatedAt invalid")
	}
	if got.UpdatedAt.Sub(currentTime) > time.Second {
		err = wrapError(err, "UpdatedAt invalid")
	}

	return err
}

// compareIDs compare if IDs exist. Do not want the same id. ID generated at site creation.
// It ignores empty and zero length wanted ids, then compares.
func compareIDs(want, got string) error {
	var err error
	var empty uuid.UUID
	parsedWant, errWant := uuid.Parse(want)
	parsedGot, errGot := uuid.Parse(got)

	// if want is empty or zero length, jut look at the got for validity
	if len(want) == 0 || parsedWant == empty {
		return errGot
	}

	if parsedWant != parsedGot {
		err = wrapError(err, "IDs not equal")
	}

	if e := compareErrors(errWant, errGot); e != nil {
		err = wrapError(err, e.Error())
	}

	return err
}

// compareErrors compares errors' nil values and messages
func compareErrors(want, got error) error {
	if want == nil && got == nil { // both are nil
		return nil
	}

	if want == nil || got == nil { // just one is nil
		return errors.New("Errors not both nil")
	}

	if want.Error() != got.Error() {
		return errors.New("Errors not equal")
	}

	return nil
}

// wrapError augments the strings.Wrap to not return nil when message is appended to a nil error
func wrapError(err error, message string) error {
	if len(strings.TrimSpace(message)) == 0 {
		return err
	}

	if err == nil {
		return errors.New(message)
	}

	return errors.Wrap(err, message)
}
