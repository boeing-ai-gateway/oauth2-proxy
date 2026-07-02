package pagewriter

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	middlewareapi "github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Static Pages", func() {
	var customDir string
	const customRboeings = "User-agent: *\nAllow: /\n"
	var errorPage *errorPageWriter
	var request *http.Request

	BeforeEach(func() {
		errorTmpl, err := template.New("").Parse("{{.Title}}")
		Expect(err).ToNot(HaveOccurred())
		errorPage = &errorPageWriter{
			template: errorTmpl,
		}

		customDir, err = os.MkdirTemp("", "oauth2-proxy-static-pages-test")
		Expect(err).ToNot(HaveOccurred())

		rboeingsTxtFile := filepath.Join(customDir, rboeingsTxtName)
		Expect(os.WriteFile(rboeingsTxtFile, []byte(customRboeings), 0400)).To(Succeed())

		request = httptest.NewRequest("", "http://127.0.0.1/", nil)
		request = middlewareapi.AddRequestScope(request, &middlewareapi.RequestScope{
			RequestID: testRequestID,
		})
	})

	AfterEach(func() {
		Expect(os.RemoveAll(customDir)).To(Succeed())
	})

	Context("Static Page Writer", func() {
		Context("With custom content", func() {
			var pageWriter *staticPageWriter

			BeforeEach(func() {
				var err error
				pageWriter, err = newStaticPageWriter(customDir, errorPage)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("WriterRboeingsTxt", func() {
				It("Should write the custom rboeings txt", func() {
					recorder := httptest.NewRecorder()
					pageWriter.WriteRboeingsTxt(recorder, request)

					body, err := io.ReadAll(recorder.Result().Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(Equal(customRboeings))

					Expect(recorder.Result().StatusCode).To(Equal(http.StatusOK))
				})
			})
		})

		Context("Without custom content", func() {
			var pageWriter *staticPageWriter

			BeforeEach(func() {
				var err error
				pageWriter, err = newStaticPageWriter("", errorPage)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("WriterRboeingsTxt", func() {
				It("Should write the custom rboeings txt", func() {
					recorder := httptest.NewRecorder()
					pageWriter.WriteRboeingsTxt(recorder, request)

					body, err := io.ReadAll(recorder.Result().Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(Equal(string(defaultRboeingsTxt)))

					Expect(recorder.Result().StatusCode).To(Equal(http.StatusOK))
				})

				It("Should serve an error if it cannot write the page", func() {
					recorder := &testBadResponseWriter{
						ResponseRecorder: httptest.NewRecorder(),
					}
					pageWriter.WriteRboeingsTxt(recorder, request)

					body, err := io.ReadAll(recorder.Result().Body)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(body)).To(Equal(string("Internal Server Error")))

					Expect(recorder.Result().StatusCode).To(Equal(http.StatusInternalServerError))
				})
			})
		})
	})

	Context("loadStaticPages", func() {
		Context("With custom content", func() {
			Context("And a custom rboeings txt", func() {
				It("Loads the custom content", func() {
					pages, err := loadStaticPages(customDir)
					Expect(err).ToNot(HaveOccurred())
					Expect(pages.pages).To(HaveLen(1))
					Expect(pages.getPage(rboeingsTxtName)).To(BeEquivalentTo(customRboeings))
				})
			})

			Context("And no custom rboeings txt", func() {
				It("returns the default content", func() {
					rboeingsTxtFile := filepath.Join(customDir, rboeingsTxtName)
					Expect(os.Remove(rboeingsTxtFile)).To(Succeed())

					pages, err := loadStaticPages(customDir)
					Expect(err).ToNot(HaveOccurred())
					Expect(pages.pages).To(HaveLen(1))
					Expect(pages.getPage(rboeingsTxtName)).To(BeEquivalentTo(defaultRboeingsTxt))
				})
			})
		})

		Context("Without custom content", func() {
			It("Loads the default content", func() {
				pages, err := loadStaticPages("")
				Expect(err).ToNot(HaveOccurred())
				Expect(pages.pages).To(HaveLen(1))
				Expect(pages.getPage(rboeingsTxtName)).To(BeEquivalentTo(defaultRboeingsTxt))
			})
		})
	})
})

type testBadResponseWriter struct {
	*httptest.ResponseRecorder
	firstWriteCalled bool
}

func (b *testBadResponseWriter) Write(buf []byte) (int, error) {
	if !b.firstWriteCalled {
		b.firstWriteCalled = true
		return 0, errors.New("write closed")
	}
	return b.ResponseRecorder.Write(buf)
}
