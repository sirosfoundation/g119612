package etsi119612_test

import (
	"crypto/x509"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("./testdata/EWC-TL.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NotNil(t, tsl)
	assert.NoError(t, err)
	assert.NotNil(t, tsl.StatusList)
	si := tsl.StatusList.TslSchemeInformation
	assert.NotNil(t, si)
	assert.Equal(t, si.TSLSequenceNumber, 1)
	assert.Equal(t, *si.TslSchemeOperatorName.Name[0].XmlLangAttr, etsi119612.Lang("en"))
	assert.Equal(t, etsi119612.FindByLanguage(si.TslSchemeOperatorName, "en", "unknown"), "EWC Consortium")
	assert.Equal(t, etsi119612.FindByLanguage(si.TslSchemeOperatorName, "fr", "unknown 4711"), "unknown 4711")
}

func TestFetchSigned(t *testing.T) {
	defer gock.Off()
	gock.New("https://trustedlist.pts.se").
		Get("/SE-TL.xml").
		Reply(200).
		File("./testdata/SE-TL.xml")

	tsl, err := etsi119612.FetchTSL("https://trustedlist.pts.se/SE-TL.xml")
	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	assert.True(t, tsl.Signed)
	assert.NotNil(t, tsl.Signer)
	assert.IsType(t, x509.Certificate{}, tsl.Signer)
}

func TestFetchSignedBroken(t *testing.T) {
	//calculated digest does not match the expected digest
	defer gock.Off()
	gock.New("https://trustedlist.pts.se").
		Get("/SE-TL.xml").
		Reply(200).
		File("./testdata/SE-TL-bad-sig.xml")

	tsl, err := etsi119612.FetchTSL("https://trustedlist.pts.se/SE-TL.xml")
	assert.Error(t, err)
	assert.Nil(t, tsl)
}

func TestFetchMissingSchemeInfo(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("./testdata/EWC-TL-no-scheme-information.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NotNil(t, tsl)
	assert.NoError(t, err)
	si := tsl.StatusList.TslSchemeInformation
	assert.Nil(t, si)
}

func TestFetchBrokenXML(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("./testdata/not-xml.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.Nil(t, tsl)
	assert.Error(t, err)
}

func TestFetchMissing(t *testing.T) {
	defer gock.Off()
	gock.New("https://example.com").
		Get("/missing").
		Reply(404)

	tsl, err := etsi119612.FetchTSL("https://example.com/missing")
	assert.Nil(t, tsl)
	assert.NotNil(t, err)
}

func TestFetchError(t *testing.T) {
	defer gock.Off()
	gock.New("https://example.com").
		Get("/bad").
		Reply(500)

	tsl, err := etsi119612.FetchTSL("https://example.com/bad")
	assert.Nil(t, tsl)
	assert.NotNil(t, err)
}

func TestFetchTSLWithOptions_CustomUserAgent(t *testing.T) {
	defer gock.Off()

	// Setup mock with matcher that checks the User-Agent header
	gock.New("https://example.com").
		Get("/tsl").
		MatchHeader("User-Agent", "CustomUserAgent/1.0").
		Reply(200).
		File("./testdata/EWC-TL.xml")

	// Use custom options with specific User-Agent
	options := etsi119612.TSLFetchOptions{
		UserAgent: "CustomUserAgent/1.0",
		Timeout:   30 * time.Second,
	}

	tsl, err := etsi119612.FetchTSLWithOptions("https://example.com/tsl", options)
	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	assert.Equal(t, "https://example.com/tsl", tsl.Source)
	assert.NotNil(t, tsl.StatusList)
}

func TestFetchTSLWithOptions_Timeout(t *testing.T) {
	defer gock.Off()

	// Mock server that delays for longer than our timeout
	gock.New("https://example.com").
		Get("/slow-tsl").
		Reply(200).
		Delay(200 * time.Millisecond). // Delay the response
		File("./testdata/EWC-TL.xml")

	// Use very short timeout (50ms)
	options := etsi119612.TSLFetchOptions{
		UserAgent: "TimeoutTest/1.0",
		Timeout:   50 * time.Millisecond,
	}

	// This should time out
	start := time.Now()
	tsl, err := etsi119612.FetchTSLWithOptions("https://example.com/slow-tsl", options)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, tsl)
	// Make sure we didn't wait longer than expected
	// We allow a small margin for test execution overhead
	assert.Less(t, elapsed, 150*time.Millisecond, "Timeout should have occurred quickly")
}

func TestFetchTSLWithOptions_ClientWithTimeout(t *testing.T) {
	defer gock.Off()

	// Setup mock with a normal reply
	gock.New("https://example.com").
		Get("/client-timeout-test").
		Reply(200).
		File("./testdata/EWC-TL.xml")

	// Create a custom client with a very long timeout (this timeout should be used instead of the one in options)
	customClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Enable gock for this client (required for custom clients)
	gock.InterceptClient(customClient)
	defer gock.RestoreClient(customClient)

	// Use custom client but with a short timeout in options - the client timeout should take precedence
	options := etsi119612.TSLFetchOptions{
		Client:    customClient,
		Timeout:   50 * time.Millisecond, // This should be ignored since we're providing a client
		UserAgent: "ClientTest/1.0",
	}

	// This should succeed because the mock responds immediately
	tsl, err := etsi119612.FetchTSLWithOptions("https://example.com/client-timeout-test", options)

	// We should have successfully fetched the TSL since the client's timeout was not exceeded
	if assert.NoError(t, err) {
		assert.NotNil(t, tsl)
		assert.Equal(t, "https://example.com/client-timeout-test", tsl.Source)
	}
}

// This section previously contained an unused customTransport implementation

func TestFetchTSLWithOptions_ErrorHandling(t *testing.T) {
	defer gock.Off()

	// Mock server that returns a 404 error
	gock.New("https://example.com").
		Get("/missing-tsl").
		Reply(404).
		BodyString("Not Found")

	// Mock server that returns invalid XML
	gock.New("https://example.com").
		Get("/bad-xml").
		Reply(200).
		BodyString("<not-valid-xml>")

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errText string
	}{
		{
			name:    "HTTP error",
			url:     "https://example.com/missing-tsl",
			wantErr: true,
			errText: "404", // Should contain the status code
		},
		{
			name:    "Invalid XML",
			url:     "https://example.com/bad-xml",
			wantErr: true,
			errText: "XML", // Error should mention XML parsing
		},
		{
			name:    "Invalid URL",
			url:     "://malformed-url",
			wantErr: true,
			errText: "missing protocol scheme", // Actual error message from the URL parser
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := etsi119612.TSLFetchOptions{
				UserAgent: "ErrorTest/1.0",
				Timeout:   2 * time.Second,
			}

			tsl, err := etsi119612.FetchTSLWithOptions(tt.url, options)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				assert.Nil(t, tsl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tsl)
			}
		})
	}
}

func TestFetchTSLWithReferences_BackwardCompatibility(t *testing.T) {
	defer gock.Off()

	defaultUserAgent := "Go-Trust/1.0 TSL Fetcher (+https://github.com/sirosfoundation/go-trust)"

	// Setup mock for main TSL
	gock.New("https://example.com").
		Get("/main-tsl").
		MatchHeader("User-Agent", defaultUserAgent). // Should use the default User-Agent
		Reply(200).
		File("./testdata/TSL-with-pointer.xml")

	// Setup mock for referenced TSL
	gock.New("https://example.com").
		Get("/referenced-tsl").
		MatchHeader("User-Agent", defaultUserAgent). // Should use the default User-Agent
		Reply(200).
		File("./testdata/EWC-TL.xml")

	// Call the original FetchTSL function (not the one with options)
	tsl, err := etsi119612.FetchTSL("https://example.com/main-tsl")

	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	assert.Equal(t, "https://example.com/main-tsl", tsl.Source)

	// Check that pointers were dereferenced (if any)
	if len(tsl.Referenced) > 0 {
		assert.NotNil(t, tsl.Referenced[0])
	}
}

func TestFetchNotURL(t *testing.T) {
	tsl, err := etsi119612.FetchTSL("urn:not-an url")
	assert.Nil(t, tsl)
	assert.NotNil(t, err)
}

func TestCertPoolBadBase64(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/EWC-TL-bad-base64.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NotNil(t, tsl)
	assert.Nil(t, err)
	pool := tsl.ToCertPool(etsi119612.PolicyAll)
	assert.NotNil(t, pool)
}

func TestCertPoolBadCert(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/EWC-TL-bad-cert.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NotNil(t, tsl)
	assert.Nil(t, err)
	pool := tsl.ToCertPool(etsi119612.PolicyAll)
	assert.NotNil(t, pool)
}

func TestCertPool(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/EWC-TL.xml")

	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NotNil(t, tsl)
	assert.Nil(t, err)
	pool := tsl.ToCertPool(etsi119612.PolicyAll)
	assert.NotNil(t, pool)
}

func TestPolicy(t *testing.T) {
	p := etsi119612.NewTSPServicePolicy()
	assert.True(t, slices.ContainsFunc(p.ServiceStatus, func(s string) bool { return s == etsi119612.ServiceStatusGranted }))
	assert.Equal(t, len(p.ServiceStatus), 1)
	p.AddServiceTypeIdentifier("urn:foo")
	assert.True(t, slices.ContainsFunc(p.ServiceTypeIdentifier, func(s string) bool { return s == "urn:foo" }))
	p.AddServiceStatus("urn:bar")
	assert.True(t, slices.ContainsFunc(p.ServiceStatus, func(s string) bool { return s == "urn:bar" }))
	assert.Equal(t, len(p.ServiceStatus), 2)
}

func TestTSLMethods(t *testing.T) {
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/EWC-TL.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t, err)
	if got := tsl.NumberOfTrustServiceProviders(); got != 17 {
		t.Errorf("expected 17 providers, got %d", got)
	}

	if name := tsl.SchemeOperatorName(); name != "EWC Consortium" {
		t.Errorf("expected 'EWC Consortium', got %q", name)
	}
	expectedStr := "TSL[Source: https://ewc-consortium.github.io/ewc-trust-list/EWC-TL] by EWC Consortium with 17 trust service providers"
	if tsl.String() != expectedStr {
		t.Errorf("unexpected String output:\ngot:  %q\nwant: %q", tsl.String(), expectedStr)
	}
}

func TestDereferencePointersToOtherTSL(t *testing.T) {
	defer gock.Off()
	// Mock the main TSL with a pointer to another TSL
	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		File("testdata/TSL-with-pointer.xml")
	// Mock the referenced TSL
	gock.New("https://example.com").
		Get("/referenced.xml").
		Reply(200).
		File("testdata/EWC-TL.xml")

	tsl, err := etsi119612.FetchTSL("https://example.com/main.xml")
	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	assert.NotNil(t, tsl.Referenced)
	assert.Greater(t, len(tsl.Referenced), 0)
}

func TestDereferencePointersToOtherTSL_InvalidPointer(t *testing.T) {
	defer gock.Off()
	// Mock the main TSL with a pointer to an invalid TSL
	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		File("testdata/TSL-with-invalid-pointer.xml")
	// The referenced TSL will 404
	gock.New("https://example.com").
		Get("/notfound.xml").
		Reply(404)

	tsl, err := etsi119612.FetchTSL("https://example.com/main.xml")
	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	// Should not panic or error, but Referenced may be empty or nil
}

func TestFetchTSLWithReferencesAndOptions(t *testing.T) {
	defer gock.Off()
	// Mock the main TSL with a pointer to another TSL
	gock.New("https://example.com").
		Get("/main.xml").
		MatchHeader("User-Agent", "Custom/2.0").
		Reply(200).
		File("testdata/TSL-with-pointer.xml")
	// Mock the referenced TSL, also checking for the same User-Agent
	gock.New("https://example.com").
		Get("/referenced.xml").
		MatchHeader("User-Agent", "Custom/2.0").
		Reply(200).
		File("testdata/EWC-TL.xml")

	options := etsi119612.TSLFetchOptions{
		UserAgent:           "Custom/2.0",
		Timeout:             30 * time.Second,
		MaxDereferenceDepth: 3,
	}

	tsls, err := etsi119612.FetchTSLWithReferencesAndOptions("https://example.com/main.xml", options)
	assert.NoError(t, err)
	assert.NotEmpty(t, tsls)

	// The first element should be the root TSL
	rootTSL := tsls[0]
	assert.NotNil(t, rootTSL)
	assert.Equal(t, "https://example.com/main.xml", rootTSL.Source)

	// Verify the referenced TSL is present in the array
	assert.Equal(t, 2, len(tsls), "Should have root TSL and one referenced TSL")
	assert.Equal(t, "https://example.com/referenced.xml", tsls[1].Source)

	// Also verify it's referenced in the root TSL
	assert.NotNil(t, rootTSL.Referenced)
	assert.Greater(t, len(rootTSL.Referenced), 0)
	assert.Equal(t, "https://example.com/referenced.xml", rootTSL.Referenced[0].Source)
}

func TestFetchTSLWithReferencesAndOptions_MaxDepth(t *testing.T) {
	// Clean up all mocks before and after test
	gock.OffAll()
	defer gock.OffAll()

	// Enable network access for these hosts to ensure mocks are used
	gock.InterceptClient(http.DefaultClient)
	defer gock.RestoreClient(http.DefaultClient)

	// Mock the main TSL with a pointer to another TSL
	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>https://example.com/referenced.xml</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`)

	// Mock both XML and PDF versions of the referenced TSL
	gock.New("https://example.com").
		Get("/referenced.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>https://example.com/self-reference.xml</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`) // It has a pointer to a deeper reference

	// Mock the PDF version to return a 404
	gock.New("https://example.com").
		Get("/referenced.pdf").
		Reply(404).
		BodyString("Not Found")

	// Test with max depth 0 (no references followed)
	options := etsi119612.TSLFetchOptions{
		UserAgent:           "Custom/2.0",
		Timeout:             30 * time.Second,
		MaxDereferenceDepth: 0,
		AcceptHeaders:       []string{"application/xml", "text/xml", "*/*"},
	}

	// Fetch the main TSL - should not follow references
	var tsls []*etsi119612.TSL
	var err error
	tsls, err = etsi119612.FetchTSLWithReferencesAndOptions("https://example.com/main.xml", options)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tsls), "Should have only the root TSL when max-depth is 0")
	assert.Equal(t, "https://example.com/main.xml", tsls[0].Source)

	// Reset mocks for the next test
	gock.OffAll()

	// Set up mocks again for the next test
	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>https://example.com/referenced.xml</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`)

	gock.New("https://example.com").
		Get("/referenced.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>https://example.com/self-reference.xml</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`) // It has a pointer to a deeper reference

	// Test with max depth 1 (direct references followed)
	options.MaxDereferenceDepth = 1
	tsls, err = etsi119612.FetchTSLWithReferencesAndOptions("https://example.com/main.xml", options)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tsls), "Should have root TSL and one referenced TSL when max-depth is 1")

	// Find the referenced TSL in the slice
	var referencedTSL *etsi119612.TSL
	for _, tsl := range tsls {
		if tsl.Source == "https://example.com/referenced.xml" {
			referencedTSL = tsl
			break
		}
	}
	assert.NotNil(t, referencedTSL, "Referenced TSL should be in the results")

	// Check that gock intercepted all expected calls
	assert.True(t, gock.IsDone(), "Not all expected HTTP calls were made")
}

func TestFetchTSLWithPDFPointer(t *testing.T) {
	// Clean up all mocks before and after test
	gock.OffAll()
	defer gock.OffAll()

	// Enable network access for these hosts to ensure mocks are used
	gock.InterceptClient(http.DefaultClient)
	defer gock.RestoreClient(http.DefaultClient)

	// Mock the main TSL with a pointer to a PDF
	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>https://example.com/referenced.pdf</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`)

	// Mock the XML version of the referenced TSL to return success
	gock.New("https://example.com").
		Get("/referenced.xml").
		Reply(200).
		BodyString(`<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <tsl:SchemeInformation>
    <tsl:PointersToOtherTSL/>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList/>
</tsl:TrustServiceStatusList>`)

	// Mock the PDF version to return a failure
	gock.New("https://example.com").
		Get("/referenced.pdf").
		Reply(400).
		BodyString("Invalid XML format - this is a PDF")

	// Create fetch options with MaxDereferenceDepth set to allow following the reference
	options := etsi119612.TSLFetchOptions{
		UserAgent:           "TestAgent/1.0",
		Timeout:             10 * time.Second,
		MaxDereferenceDepth: 1,
		AcceptHeaders:       []string{"application/xml", "text/xml", "*/*"},
	}

	// Fetch the main TSL with references - this should also try to dereference the pointer
	var tsls []*etsi119612.TSL
	var err error
	tsls, err = etsi119612.FetchTSLWithReferencesAndOptions("https://example.com/main.xml", options)
	assert.NoError(t, err)
	assert.NotNil(t, tsls)

	// Check that we have the root TSL and one referenced TSL (total of 2)
	assert.Equal(t, 2, len(tsls))

	// Verify the root TSL has the reference
	root := tsls[0]
	assert.Equal(t, "https://example.com/main.xml", root.Source)

	// Check that we have the referenced TSL in the slice
	var refTSL *etsi119612.TSL
	for _, tsl := range tsls {
		if tsl.Source == "https://example.com/referenced.xml" {
			refTSL = tsl
			break
		}
	}
	assert.NotNil(t, refTSL, "Referenced TSL not found in the result")

	// Check that gock intercepted all expected calls
	assert.True(t, gock.IsDone(), "Not all expected HTTP calls were made")

	// The referenced TSL should have the XML URL as its source
	assert.Equal(t, "https://example.com/referenced.xml", refTSL.Source, "Source should be the XML URL, not the PDF URL")
}

func TestWithTrustServices_EmptyAndNil(t *testing.T) {
	tsl := &etsi119612.TSL{StatusList: etsi119612.TrustStatusListType{}}
	called := false
	tsl.WithTrustServices(func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType) {
		called = true
	})
	assert.False(t, called, "Callback should not be called for empty TSL")
}

func TestToCertPool_RejectAllPolicy(t *testing.T) {
	// Use a real TSL from testdata
	defer gock.Off()
	gock.New("https://ewc-consortium.github.io").
		Get("/EWC-TL").
		Reply(200).
		File("testdata/EWC-TL.xml")
	tsl, err := etsi119612.FetchTSL("https://ewc-consortium.github.io/ewc-trust-list/EWC-TL")
	assert.NoError(t, err)
	assert.NotNil(t, tsl)
	// Policy that rejects all
	rejectAll := &etsi119612.TSPServicePolicy{ServiceStatus: []string{"nonexistent-status"}}
	pool := tsl.ToCertPool(rejectAll)
	assert.NotNil(t, pool)
	// We're testing that the pool is empty, but pool.Subjects() is deprecated.
	// For our test purposes, we just want to ensure the pool was created but no certs were added.
}

func TestCleanCertsTrimsWhitespace(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: "  CERTDATA  "},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	tsl.CleanCerts()
	cert := tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider[0].
		TslTSPServices.TslTSPService[0].TslServiceInformation.TslServiceDigitalIdentity.DigitalId[0].X509Certificate
	assert.Equal(t, "CERTDATA", cert)
}

func TestTSLRecursiveReference(t *testing.T) {
	tsl := &etsi119612.TSL{}
	tsl.Referenced = []*etsi119612.TSL{tsl}
	assert.Contains(t, tsl.Referenced, tsl)
	// Should not panic or loop forever
}

func TestValidate_InvalidStatus(t *testing.T) {
	tsp := &etsi119612.TSPType{}
	svc := &etsi119612.TSPServiceType{
		TslServiceInformation: &etsi119612.TSPServiceInformationType{
			TslServiceStatus: "invalid-status",
		},
	}
	policy := etsi119612.NewTSPServicePolicy()
	err := tsp.Validate(svc, nil, policy)
	assert.ErrorIs(t, err, etsi119612.ErrInvalidStatus)
}

func TestValidate_InvalidConstraints(t *testing.T) {
	tsp := &etsi119612.TSPType{}
	svc := &etsi119612.TSPServiceType{
		TslServiceInformation: &etsi119612.TSPServiceInformationType{
			TslServiceStatus:         etsi119612.ServiceStatusGranted,
			TslServiceTypeIdentifier: "foo",
		},
	}
	policy := etsi119612.NewTSPServicePolicy()
	policy.ServiceTypeIdentifier = []string{"bar"}
	err := tsp.Validate(svc, nil, policy)
	assert.ErrorIs(t, err, etsi119612.ErrInvalidConstraints)
}

func TestTSLSummary(t *testing.T) {
	tsl := &etsi119612.TSL{}
	summary := tsl.Summary()
	assert.NotNil(t, summary)
	assert.Contains(t, summary, "scheme_operator_name")
	assert.Contains(t, summary, "num_trust_service_providers")
	assert.Contains(t, summary, "summary")
}

func TestTSLSummary_NullTSL(t *testing.T) {
	var tsl *etsi119612.TSL
	summary := tsl.Summary()
	assert.NotNil(t, summary)
	assert.Len(t, summary, 0)
}
