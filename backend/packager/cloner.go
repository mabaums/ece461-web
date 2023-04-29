package packager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	b64 "encoding/base64"

	"github.com/mabaums/ece461-web/backend/models"
	log "github.com/sirupsen/logrus"
)

type PackageJSON struct {
	Name    string
	Version string
}

func GetPackageJson(url string) (*models.PackageMetadata, string, error) {
	tempDir, err := os.MkdirTemp(".", "*")

	if err != nil {
		log.Errorf("Error creating temporary folder %v", err)
		return nil, "", err
	}
	defer os.RemoveAll(tempDir)

	err = Clone(tempDir, url)
	if err != nil {
		return nil, "", err
	}

	metadata, err := ReadPackageJson(tempDir)
	if err != nil {
		return nil, "", err
	}

	encoded, err := zipEncodeDir(tempDir)
	if err != nil {
		return nil, "", err
	}

	// check for errors here.

	return metadata, encoded, err
}

func zipEncodeDir(dir string) (string, error) {
	file, err := os.Create("output.zip")
	if err != nil {
		log.Errorf("Error creating output.zip file")
		return "", err
	}

	w := zip.NewWriter(file)

	walker := func(path string, info os.FileInfo, err error) error {
		//fmt.Printf("Crawling: %#v\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Ensure that `path` is not absolute; it should not start with "/".
		// This snippet happens to work because I don't use
		// absolute paths, but ensure your real-world code
		// transforms path into a zip-root relative path.
		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(dir, walker)
	if err != nil {
		log.Errorf("Error walking and creating zip from file %v", err)
		return "", err
	}
	w.Close()
	file.Close()
	bytes, err := os.ReadFile("output.zip")
	if err != nil {
		log.Errorf("Error reading output zip %v", err)
		return "", err
	}
	sEnc := b64.StdEncoding.EncodeToString(bytes)
	os.RemoveAll("output.zip")
	return sEnc, nil

}

func Clone(dir string, url string) error {
	log.Infof("Cloning %v into %v", url, dir)
	cmd := exec.Command("git", "clone", url, dir)
	err := cmd.Run()
	if err != nil {
		log.Errorf("Error Cloning: %v in Dir: %v Err: %v", url, dir, err) // Maybe no need to be Fatal?
	}
	return err
}

func Rate(url string) (*models.PackageRating, error) {
	log.Infof("Rating pacakge %v", url)
	f, err := os.CreateTemp(".", "*")
	if err != nil {
		log.Errorf("Error creating temporary file for rate. %v", err)
		return nil, err
	}
	defer os.RemoveAll(f.Name())

	_, err = f.WriteString(url)
	if err != nil {
		log.Errorf("Error writing string (%v) to temp file: %v Err: %v", url, f.Name(), err)
		return nil, err
	}
	f.Close()
	cmd := exec.Command("./cli", f.Name())
	r_Bytes, err := cmd.Output()
	if err != nil {
		log.Errorf("Error obtaining output from command: %v", err)
		return nil, err
	}
	var ratingMap map[string]interface{}
	err = json.Unmarshal(r_Bytes, &ratingMap)
	if err != nil {
		log.Errorf("Error unmarshaling json: %v", err)
	}

	ratings := models.PackageRating{
		RampUp:               ratingMap["RAMP_UP_SCORE"].(float64),
		BusFactor:            ratingMap["BUS_FACTOR_SCORE"].(float64),
		Correctness:          ratingMap["CORRECTNESS_SCORE"].(float64),
		LicenseScore:         ratingMap["LICENSE_SCORE"].(float64),
		NetScore:             ratingMap["NET_SCORE"].(float64),
		ResponsiveMaintainer: ratingMap["RESPONSIVE_MAINTAINER_SCORE"].(float64),
		GoodPinningPractice:  ratingMap["VERSION_PINNING_SCORE"].(float64),
	}

	fmt.Printf("%+v", ratings)

	return &ratings, nil
}

func ReadPackageJson(dir string) (*models.PackageMetadata, error) {
	log.Infof("Reading package.json in %v", dir)
	var metadata models.PackageMetadata

	content, err := ioutil.ReadFile(filepath.Join(dir, "package.json"))

	if err != nil {
		log.Errorf("No package.json found in %v", dir)
		return nil, err
	}

	var pkgJSON PackageJSON

	err = json.Unmarshal(content, &pkgJSON)

	if err != nil {
		log.Errorf("package.json is invalid: %v", err)
		return nil, err
	}
	log.Infof("Parsed package.json %+v", pkgJSON)

	metadata = models.PackageMetadata{
		Name:    pkgJSON.Name,
		Version: pkgJSON.Version,
	}

	return &metadata, nil
}
