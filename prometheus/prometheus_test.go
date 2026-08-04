package prometheus

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tests for Prometheus", func() {

	Context("Tests for RoundTrip()", func() {
		var bat authTransport
		BeforeEach(func() {
			bat.username = "someRandomUsername"
			bat.password = "someRandomPassword"
			bat.token = "someRandomToken"
			bat.Transport = http.DefaultTransport
			count = 0
		})

		It("Test1 for default behaviour", func() {
			url := "https://example.com/api"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				fmt.Println("Failed to create request:", err)
				return
			}
			_, err = bat.RoundTrip(req)
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err).To(BeNil())
		})

		It("Test2 bearer header is used when token is provided", func() {
			url := "https://example.com/api"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				fmt.Println("Failed to create request:", err)
				return
			}
			_, err = bat.RoundTrip(req)
			Expect(req.Header.Get("Authorization")).To(Equal("Bearer someRandomToken"))
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err).To(BeNil())
		})

		It("Test3 basic auth header is used when no token is provided", func() {
			bat.token = ""
			url := "https://example.com/api"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				fmt.Println("Failed to create request:", err)
				return
			}
			_, err = bat.RoundTrip(req)

			encodedAuthHeader := base64.StdEncoding.EncodeToString([]byte("someRandomUsername:someRandomPassword"))

			Expect(req.Header.Get("Authorization")).To(Equal("Basic " + encodedAuthHeader))
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err).To(BeNil())
		})

		It("Test4 no auth header set when auth details are omitted", func() {
			bat.token = ""
			bat.username = ""
			bat.password = ""
			url := "https://example.com/api"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				fmt.Println("Failed to create request:", err)
				return
			}
			_, err = bat.RoundTrip(req)

			Expect(req.Header.Get("Authorization")).To(Equal(""))
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err).To(BeNil())
		})
	})

	Context("Tests for RoundTrip() with tokenFile", func() {
		var tmpDir string
		var tokenFilePath string
		var noopTransport http.RoundTripper

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "token-test-*")
			Expect(err).NotTo(HaveOccurred())
			tokenFilePath = tmpDir + "/token"
			noopTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200}, nil
			})
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		It("reads token from file on each request", func() {
			Expect(os.WriteFile(tokenFilePath, []byte("token-v1"), 0600)).To(Succeed())
			bat := authTransport{tokenFile: tokenFilePath, Transport: noopTransport}

			req, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err := bat.RoundTrip(req)
			Expect(err).To(BeNil())
			Expect(req.Header.Get("Authorization")).To(Equal("Bearer token-v1"))

			Expect(os.WriteFile(tokenFilePath, []byte("token-v2"), 0600)).To(Succeed())
			req2, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err = bat.RoundTrip(req2)
			Expect(err).To(BeNil())
			Expect(req2.Header.Get("Authorization")).To(Equal("Bearer token-v2"))
		})

		It("trims whitespace from token file contents", func() {
			Expect(os.WriteFile(tokenFilePath, []byte("  my-token \n\n"), 0600)).To(Succeed())
			bat := authTransport{tokenFile: tokenFilePath, Transport: noopTransport}

			req, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err := bat.RoundTrip(req)
			Expect(err).To(BeNil())
			Expect(req.Header.Get("Authorization")).To(Equal("Bearer my-token"))
		})

		It("returns error when token file does not exist", func() {
			bat := authTransport{tokenFile: "/nonexistent/token", Transport: noopTransport}
			req, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err := bat.RoundTrip(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read token file"))
		})

		It("does not set Authorization when token file is empty", func() {
			Expect(os.WriteFile(tokenFilePath, []byte(""), 0600)).To(Succeed())
			bat := authTransport{tokenFile: tokenFilePath, Transport: noopTransport}

			req, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err := bat.RoundTrip(req)
			Expect(err).To(BeNil())
			Expect(req.Header.Get("Authorization")).To(Equal(""))
		})

		It("tokenFile takes precedence over static token", func() {
			Expect(os.WriteFile(tokenFilePath, []byte("file-token"), 0600)).To(Succeed())
			bat := authTransport{token: "static-token", tokenFile: tokenFilePath, Transport: noopTransport}

			req, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
			_, err := bat.RoundTrip(req)
			Expect(err).To(BeNil())
			Expect(req.Header.Get("Authorization")).To(Equal("Bearer file-token"))
		})
	})

	Context("Tests for NewClient()", func() {
		var url, username, password, token string
		var tlsSkipVerify bool
		BeforeEach(func() {
			url = ""
			username = ""
			password = ""
			token = ""
			tlsSkipVerify = false
			count = 0
		})
		It("Test1 empty parameters", func() {
			_, err := NewClient(url, token, username, password, tlsSkipVerify)
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err.Error()).To(ContainSubstring("Post \"/api/v1/query\": unsupported protocol scheme \"\""))
		})

		It("Test2 passing not valid url", func() {
			url = "not a valid url:port"
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			_, err := NewClient(url, token, username, password, tlsSkipVerify)
			Expect(err.Error()).To(ContainSubstring("parse \"not a valid url:port\": first path segment in URL cannot contain colon"))
		})

	})

	Context("Tests for Query()", func() {
		var url, username, password, token string
		var tlsSkipVerify bool
		var pr *Prometheus
		BeforeEach(func() {
			pr, _ = NewClient(url, token, username, password, tlsSkipVerify)
			count = 0
		})

		It("Test1 empty url", func() {
			_, err := pr.Query("_all", time.Now())
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err.Error()).To(ContainSubstring("Post \"/api/v1/query\": unsupported protocol scheme \"\""))
		})

		It("Test2 mock error to nil", func() {
			mockAPI := new(MockAPI)
			query := "your_query"
			start := time.Now()
			p := Prometheus{api: mockAPI}
			_, err := p.Query(query, start)
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(1))
			Expect(err).To(BeNil())
		})
	})

	Context("Tests for QueryRange()", func() {
		var url, username, password, token string
		var tlsSkipVerify bool
		var pr *Prometheus
		BeforeEach(func() {
			pr, _ = NewClient(url, token, username, password, tlsSkipVerify)
			count = 0
		})

		It("Test1 empty url", func() {
			_, err := pr.QueryRange("_all", time.Now(), time.Now().Add(time.Duration(10)), time.Duration(5))
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(0))
			Expect(err.Error()).To(ContainSubstring("Post \"/api/v1/query_range\": unsupported protocol scheme \"\""))
		})

	})

	Context("Tests for verifyConnection()", func() {
		var mockAPI *MockAPI
		var p Prometheus
		BeforeEach(func() {
			mockAPI = new(MockAPI)
			p = Prometheus{api: mockAPI}
			count = 0
		})
		It("Test1 mock to no nil", func() {
			err := p.verifyConnection()
			//Asserting no of times mocks are called
			Expect(count).To(BeEquivalentTo(1))
			Expect(err).To(BeNil())

		})
	})
})
