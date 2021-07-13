package cryptography

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/jaeha-choi/Proj_Coconut_Utility/log"
	"io/ioutil"
	"os"
)

// createRSAKey creates RSA keys with bitSize.
func createRSAKey(bitSize int) (privKey *rsa.PrivateKey, err error) {
	reader := rand.Reader
	if privKey, err = rsa.GenerateKey(reader, bitSize); err != nil {
		log.Debug(err)
		log.Error("Error while creating RSA key")
		return nil, err
	}
	return privKey, nil
}

// OpenKeys open existing keys and return them as *pem.Block.
// If keys are not found, new keys will be created.
func OpenKeys() (pubBlock *pem.Block, privBlock *pem.Block, err error) {
	var newKeyBitSize = 4096
	pubFileN := "key.pub"
	privFileN := "key.priv"

	// Check if keys already exist
	if _, err := os.Stat(pubFileN); !os.IsNotExist(err) {

		// Read existing pubkey file
		pubFileBytes, err := ioutil.ReadFile(pubFileN)
		if err != nil {
			log.Debug(err)
			log.Error("Error while opening RSA pubkey in OpenKeys")
			return nil, nil, err
		}
		// Decode file and create block
		pubBlock, _ = pem.Decode(pubFileBytes)
		if pubBlock == nil {
			log.Error("Error while decoding pubkey file")
			return nil, nil, err
		}

		// Read existing privkey file
		privFileBytes, err := ioutil.ReadFile(privFileN)
		if err != nil {
			log.Debug(err)
			log.Error("Error while opening RSA privfile in OpenKeys")
			return nil, nil, err
		}
		// Decode file and create block
		privBlock, _ = pem.Decode(privFileBytes)
		if privBlock == nil {
			log.Error("Error while decoding priv file")
			return nil, nil, err
		}
	} else if os.IsNotExist(err) {
		log.Debug(err)
		// Create new RSA key pair
		key, err := createRSAKey(newKeyBitSize)
		if err != nil {
			log.Debug(err)
			log.Error("Error while creating RSA key in OpenKeys")
			return nil, nil, err
		}
		// Open pubkey file
		pubOut, err := os.Create(pubFileN)
		if err != nil {
			log.Debug(err)
			log.Error("Error while creating pubkey file")
			return nil, nil, err
		}
		// Close pubkey file when done
		defer func() {
			if errDef := pubOut.Close(); errDef != nil {
				log.Error(err)
				log.Error("Error while closing pubkey file")
				err = errDef
				return
			}
			if errDef := os.Chmod(pubFileN, 0644); errDef != nil {
				log.Debug(errDef)
				log.Error("Error while updating pubkey file permissions")
				err = errDef
				return
			}
		}()
		// Create PEM block
		pubBlock = &pem.Block{
			Type:    "RSA PUBLIC KEY",
			Headers: nil,
			Bytes:   x509.MarshalPKCS1PublicKey(&key.PublicKey),
		}
		// Write PEM block to pubkey file
		if err := pem.Encode(pubOut, pubBlock); err != nil {
			log.Debug(err)
			log.Error("Error while writing to pubkey file")
			return nil, nil, err
		}

		// Create privkey file
		privOut, err := os.OpenFile(privFileN, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			log.Debug(err)
			log.Error("Error while creating private key in OpenKeys")
			return nil, nil, err
		}
		// Close privkey file when done
		defer func() {
			if errDef := privOut.Close(); errDef != nil {
				log.Debug(errDef)
				log.Error("Error while closing private key file")
				err = errDef
				return
			}
			if errDef := os.Chmod(privFileN, 0600); errDef != nil {
				log.Debug(errDef)
				log.Error("Error while updating permissions")
				err = errDef
				return
			}
		}()
		// Create PEM block
		privBlock = &pem.Block{
			Type:    "RSA PRIVATE KEY",
			Headers: nil,
			Bytes:   x509.MarshalPKCS1PrivateKey(key),
		}
		// Write PEM block to privkey file
		if err := pem.Encode(privOut, privBlock); err != nil {
			log.Debug(err)
			log.Error("Error while encoding private key")
			return nil, nil, err
		}
	} else {
		log.Debug(err)
		log.Error("Error while checking stat of the file")
	}
	return pubBlock, privBlock, err
}

// PemToKeys convert private PEM block to rsa.PrivateKey struct.
// *rsa.PrivateKey contains public key as well, so access public key
// with key.PublicKey
func PemToKeys(privBlock *pem.Block) (privKey *rsa.PrivateKey, err error) {
	if privKey, err = x509.ParsePKCS1PrivateKey(privBlock.Bytes); err != nil {
		log.Debug(err)
		log.Error("Error while converting PEM block to keys")
		return nil, err
	}
	return privKey, nil
}

// PemToSha256 generates bytes containing sha256sum of pubBlock bytes.
func PemToSha256(pubBlock *pem.Block) []byte {
	// sha256sum always returns 32 bytes
	hash := sha256.Sum256(pubBlock.Bytes)
	return hash[:]
}

// genSymKey generates random key for symmetric encryption
func genSymKey() (key []byte, err error) {
	// Since we're using AES, generate 32 bytes key for AES256
	key = make([]byte, 32)
	// Create random key for symmetric encryption
	if _, err := rand.Read(key); err != nil {
		log.Debug(err)
		log.Error("Error while generating symmetric encryption key")
		return nil, err
	}
	return key, nil
}

// encryptSignSymKey encrypts key for symmetric encryption with receiver's pubic key,
// and sign hashed symmetric encryption key with sender's private key.
func encryptSignSymKey(symKey []byte, receiverPubKey *rsa.PublicKey, senderPrivKey *rsa.PrivateKey, additional []byte) (
	encryptedData []byte, dataSignature []byte, err error) {
	rng := rand.Reader
	data := append(symKey, additional...)
	// Encrypt symmetric encryption key
	if encryptedData, err = rsa.EncryptOAEP(sha256.New(), rng, receiverPubKey, data, nil); err != nil {
		log.Debug(err)
		log.Error("Error while encrypting symmetric encryption key")
		return nil, nil, err
	}

	// Sign symmetric encryption key
	hashedKey := sha256.Sum256(data)
	if dataSignature, err = rsa.SignPSS(rng, senderPrivKey, crypto.SHA256, hashedKey[:], nil); err != nil {
		log.Debug(err)
		log.Error("Error while signing symmetric encryption key")
		return nil, nil, err
	}

	return encryptedData, dataSignature, nil
}

// decryptVerifySymKey decrypts key for symmetric encryption with receiver's private key,
// and verify signature with sender's public key.
func decryptVerifySymKey(encrypted []byte, signature []byte, senderPubKey *rsa.PublicKey, recieverPrivKey *rsa.PrivateKey) (
	symKey []byte, err error) {
	rng := rand.Reader

	// Decrypt symmetric encryption key
	if symKey, err = rsa.DecryptOAEP(sha256.New(), rng, recieverPrivKey, encrypted, nil); err != nil {
		log.Debug(err)
		log.Error("Error while decrypting symmetric encryption key")
		return nil, err
	}

	// Verify symmetric encryption key signature
	hashedKey := sha256.Sum256(symKey)
	if err := rsa.VerifyPSS(senderPubKey, crypto.SHA256, hashedKey[:], signature, nil); err != nil {
		log.Debug(err)
		log.Error("Invalid symmetric encryption key signature")
		return nil, err
	}

	return symKey, nil
}

// BytesToBase64 encodes raw bytes to base64
func BytesToBase64(data []byte) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data[:])
	return encoded
}