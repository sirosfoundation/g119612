package etsi119612_test

import (
	"testing"
	"os"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"encoding/json"
	"fmt"
	"encoding/base64"
	"crypto/x509"
)

type JWTCertBundle struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	X5c []string `json:"x5c"`

}

func TestLeafRootCertVerificationSuccess(t *testing.T) {
	header_mock, err:= os.ReadFile("./testdata/x5c-test-root-leaf.json")
	if err != nil {
		t.Fatalf("Failed while reading json: %v", err)
	}
    assert.NotEmpty(t, header_mock) 
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	if err !=nil{
		t.Fatalf("Failed updating jwt bundle")
	}
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg) 
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-no-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t,err)
	policy := *etsi119612.PolicyAll 
	policy.AddServiceTypeIdentifier("http://uri.etsi.org/TrstSvc/Svctype/CA/QC")
	pool := tsl.ToCertPool(&policy)
	fmt.Println("Number of trusted roots:", len(pool.Subjects()))
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots: pool})
	if err !=nil {
		t.Errorf("Chain verification failed %v", err)
	} else {
		fmt.Println("Chain verification succeeded")
	}
}

func TestLeafIntermediateRootCertVerificationSuccess(t *testing.T) {
	header_mock, err:= os.ReadFile("./testdata/x5c-test.json")
	if err != nil {
		t.Fatalf("Failed while reading json: %v", err)
	}
	assert.NoError(t, err)

    assert.NotEmpty(t, header_mock) 
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	if err !=nil{
		t.Fatalf("Failed updating jwt bundle")
	}
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg) 
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-no-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
    assert.NoError(t,err)
	policy := *etsi119612.PolicyAll 
	policy.AddServiceTypeIdentifier("http://uri.etsi.org/TrstSvc/Svctype/CA/QC")
	pool := tsl.ToCertPool(&policy)
	fmt.Println("Number of trusted roots:", len(pool.Subjects()))
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	interDER, err := base64.StdEncoding.DecodeString(jwt.X5c[1])
    assert.NoError(t, err, "Failed to decode intermediate")
    interCert, err := x509.ParseCertificate(interDER)
	assert.NoError(t, err, "Failed to parse intermediate certificate")
	intermediatePool :=x509.NewCertPool()
	intermediatePool.AddCert(interCert)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots:pool, Intermediates:intermediatePool})
	if err !=nil {
		t.Errorf("Chain verification failed %v", err)
	} else {
		fmt.Println("Chain verification succeeded")
	}
}

func TestLeafRootCertVerificationSuccessEmptyServiceTypeIdentifier(t *testing.T) {
	header_mock, err:= os.ReadFile("./testdata/x5c-test-root-leaf.json")
	assert.NoError(t, err)
    assert.NotEmpty(t, header_mock) 
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	assert.NoError(t, err, "Failed to unmarshal JWT bundle")
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg) 
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-no-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t,err)
	pool := tsl.ToCertPool(etsi119612.PolicyAll)
	fmt.Println("Number of trusted roots:", len(pool.Subjects()))
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots: pool})
	if err !=nil {
		t.Errorf("Chain verification failed %v", err)
	} else {
		fmt.Println("Chain verification succeeded")
	}
}

func TestLeafRootCertVerificationSuccessTLWithSignature(t *testing.T) {
	header_mock, err:= os.ReadFile("./testdata/x5c-test-root-leaf.json")
	assert.NoError(t, err)
    
	assert.NotEmpty(t, header_mock) 
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	assert.NoError(t, err, "Failed to unmarshal JWT bundle")
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg) 
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-with-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t,err)
	pool := tsl.ToCertPool(etsi119612.PolicyAll)
	fmt.Println("Number of trusted roots:", len(pool.Subjects()))
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots: pool})
	if err !=nil {
		t.Errorf("Chain verification failed %v", err)
	} else {
		fmt.Println("Chain verification succeeded")
	}
}

func TestServiceStatusOtherThanGrantedStatusError(t *testing.T){
	header_mock, err:= os.ReadFile("./testdata/x5c-test-root-leaf.json")
	assert.NoError(t, err)
	assert.NotEmpty(t, header_mock)
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	assert.NoError(t, err, "Failed to unmarshal JWT bundle")
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg)
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-with-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	policy := *etsi119612.PolicyAll
	policy.AddServiceStatus("https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/other-than-granted/")
	fmt.Println(policy.ServiceStatus)
	//keep only other-than-granted in the slice
	if len(policy.ServiceStatus) > 0 {
    policy.ServiceStatus = policy.ServiceStatus[1:]}
	fmt.Println("Status to test:", policy.ServiceStatus)
	pool:= tsl.ToCertPool(&policy)
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots: pool})
	assert.Error(t,err, "status is not recognized or granted")
}
func TestServiceStatusOneOfInTheListSuccess(t *testing.T){
	header_mock, err:= os.ReadFile("./testdata/x5c-test-root-leaf.json")
	assert.NoError(t, err)
	assert.NotEmpty(t, header_mock)
	var jwt JWTCertBundle
	err =json.Unmarshal(header_mock, &jwt)
	assert.NoError(t, err, "Failed to unmarshal JWT bundle")
	assert.NotEmpty(t, jwt)
	assert.NotEmpty(t, jwt.Alg)
	assert.NotEmpty(t, jwt.Typ)
	assert.NotEmpty(t, jwt.X5c)
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/test-trust-list-with-sig.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t, err)
	policy := *etsi119612.PolicyAll
	policy.AddServiceStatus("https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/other-than-granted/")
	fmt.Println(policy.ServiceStatus)
	pool:= tsl.ToCertPool(&policy)
	assert.NotNil(t, pool)
	leafDER, err:= base64.StdEncoding.DecodeString(jwt.X5c[0])
	assert.NoError(t,err)
	leafCert, err := x509.ParseCertificate(leafDER)
	assert.NoError(t,err)
	_, err = leafCert.Verify(x509.VerifyOptions{
    Roots: pool})
	if err !=nil {
		t.Errorf("Chain verification failed %v", err)
	} else {
		fmt.Println("Chain verification succeeded")
	}
}
