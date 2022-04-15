package dockerservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

var docker Docker

func TestMain(m *testing.M) {
	fmt.Println("INFO: setting up things for testing dockerservice package")
	var err error
	docker, err = NewDocker()
	if err != nil {
		log.Fatalf("couldn't create a new docker instance")
	}
	// TODO: (viggy) we should create some customer spinup images for testing purpose instead of using docker registry postgres images
	imagesToRemove := []string{"postgres:14-alpine", "postgres:13-alpine"}
	removeImageHelper(imagesToRemove)
	exitVal := m.Run()
	removeImageHelper(imagesToRemove)
	os.Exit(exitVal)
}

func removeImageHelper(imagesToRemove []string) {
	for _, imageToRemove := range imagesToRemove {
		err := removeDockerImage(docker, imageToRemove)
		if err != nil {
			log.Printf("INFO: error removing docker image %s %v \n", imageToRemove, err)
		}
	}
}

func Test_imageExistsLocally(t *testing.T) {
	data := []struct {
		name                        string
		image                       string
		pullImageFromDockerRegistry bool
		expected                    bool
	}{
		{"image exist", "postgres:14-alpine", true, true},
		{"image doesnot exist", "imageDoesnotExist:notag", false, false},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			if d.pullImageFromDockerRegistry {
				log.Println("INFO: pulling docker image from docker registry:", d.image)
				// INFO: not sure what's the best way to make sure an image exists locally. Hence pulling it before testing imageExistsLocally.
				// Perhaps we could move this to TestMain() which means we need to define a type for struct - not sure its that the right way to do
				// postgres:9.6-alpine image will be pulled since its fairly small. It could be any image.
				if err := pullImageFromDockerRegistry(docker, d.image); err != nil {
					t.Errorf("error setting up imageExistsLocally() for test data %+v", d)
				}
			}
			actual, err := imageExistsLocally(context.Background(), docker, d.image)
			if err != nil {
				t.Errorf("error testing imageExistsLocally() for test data %+v", d)
			}
			if actual != d.expected {
				t.Errorf("incorrect result: actual %t , expected %t", actual, d.expected)
			}
		})
	}
}

func Test_pullImageFromDockerRegistry(t *testing.T) {
	data := []struct {
		name     string
		image    string
		expected error
	}{
		{"image exist", "postgres:13-alpine", nil},
		{"image doesnot exist", "imageDoesnotExistInRegistry:notag", errors.New("unable to pull docker image")},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			actual := pullImageFromDockerRegistry(docker, d.image)
			if actual != d.expected {
				if !strings.Contains(actual.Error(), d.expected.Error()) {
					t.Errorf("incorrect result: actual %t , expected %t", actual, d.expected)
				}
			}
		})
	}
}
