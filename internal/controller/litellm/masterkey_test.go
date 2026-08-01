package litellm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("MasterKey Generation", func() {
	Context("buildSecretData", func() {
		It("should generate a new master key with sk- prefix when none is provided and no existing secret exists", func() {
			data := buildSecretData("", nil)
			masterKey, exists := data[masterKeyDataKey]
			Expect(exists).To(BeTrue())
			Expect(string(masterKey)).To(HavePrefix("sk-"))
			// Verify it's sk- followed by a UUID (simplified check)
			Expect(len(string(masterKey))).To(BeNumerically(">", 3))
		})

		It("should preserve existing master key if it exists in data", func() {
			existingKey := []byte("existing-key")
			existingSecret := &corev1.Secret{
				Data: map[string][]byte{
					masterKeyDataKey: existingKey,
				},
			}
			data := buildSecretData("", existingSecret)
			masterKey, exists := data[masterKeyDataKey]
			Expect(exists).To(BeTrue())
			Expect(masterKey).To(Equal(existingKey))
		})

		It("should use provided master key and not add sk- prefix if it's already provided", func() {
			providedKey := "user-provided-key"
			data := buildSecretData(providedKey, nil)
			masterKey, exists := data[masterKeyDataKey]
			Expect(exists).To(BeTrue())
			Expect(string(masterKey)).To(Equal(providedKey))
		})

		It("should generate a new master key with sk- prefix if existing secret lacks masterkey", func() {
			existingSecret := &corev1.Secret{
				Data: map[string][]byte{
					"other-key": []byte("other-value"),
				},
			}
			data := buildSecretData("", existingSecret)
			masterKey, exists := data[masterKeyDataKey]
			Expect(exists).To(BeTrue())
			Expect(string(masterKey)).To(HavePrefix("sk-"))
		})
	})
})
