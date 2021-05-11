package utils

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
)

func TestRegisterLicense(t *testing.T) {
	// Skip Certificate Check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	controllerIP := os.Getenv("PUBLIC_IP")
	var controllerURL = "https://" + controllerIP + "/v1/api"
	license := os.Getenv("AVX_LICENSE")
	password := os.Getenv("AVX_PASSWORD")
	client, err := goaviatrix.NewClient("admin", password, controllerIP, &http.Client{Transport: tr})
	if err != nil {
		t.Errorf("Expected no error, got error: %v", err)
	}
	if err = RegisterLicense(client, license, controllerURL); err != nil {
		t.Errorf("Expected no error, got errof: %v", err)
	}
}

func TestTerraform(t *testing.T) {
	strBranchName := "no_remote_state"
	gitURL := "https://github.com/vfiftyfive/terraform_aviatrix_new_hire"
	// varFilePath := os.Getenv("TF_VARFILE")
	//Install Terraform
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		panic(err)
	}
	// defer os.RemoveAll(tmpDir)
	execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	t.Logf("execPath is: %v", execPath)

	//Pull repo to be used as Terraform source
	gitDir := tmpDir + "/clone"
	_, err = git.PlainClone(gitDir, false, &git.CloneOptions{
		URL:           gitURL,
		ReferenceName: plumbing.NewBranchReferenceName(strBranchName),
		SingleBranch:  true,
	})
	if err != nil {
		t.Errorf("Expected no error, but got error: %v", err)
	}

	//Define new Terraform Structure
	workingDir := tmpDir + "/clone"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	tf.SetStdout(os.Stdout)
	t.Logf("working Directory is: %v", workingDir)

	//Run Terraform init
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// //Apply Terraform configuration
	// err = tf.Apply(context.Background(), tfexec.VarFile(varFilePath))
	// if err != nil {
	// 	t.Errorf("Expected no error, but got: %v", err)
	// }
}
