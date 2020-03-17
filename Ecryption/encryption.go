package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"math/big"

	"golang.org/x/crypto/sha3"
)

//hash hashes a slice with SHA3 256 bit hash algorithm
func hash(data []byte) []byte {
	hashArray := sha3.Sum256(data)
	return hashArray[:]
}

//Encryptor keeps required data for encryption of respective communication channel
type Encryptor struct {
	curve      elliptic.Curve
	privateKey []byte
	publicKeyX *big.Int
	publicKeyY *big.Int
	sharedKey  []byte
	aesBlock   cipher.Block
	gcm        cipher.AEAD
}

//New creates and initializes an Encryptor object
func New() (*Encryptor, error) {
	curve := elliptic.P256()
	privateKey, publicKeyX, publicKeyY, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Encryptor{curve: curve, privateKey: privateKey, publicKeyX: publicKeyX, publicKeyY: publicKeyY}, nil
}

//Prepare gets the public key of the communication peer and prepare the respective Encryptor object
//for encryption of the communication channel
func (enc *Encryptor) Prepare(peerPublicKey []byte) error {
	X, Y := elliptic.Unmarshal((*enc).curve, peerPublicKey)
	X, Y = (*enc).curve.ScalarMult(X, Y, (*enc).privateKey)
	(*enc).sharedKey = hash(elliptic.Marshal((*enc).curve, X, Y))
	aesBlock, err := aes.NewCipher((*enc).sharedKey)
	(*enc).aesBlock = aesBlock
	if err != nil {
		return err
	}
	(*enc).gcm, err = cipher.NewGCM((*enc).aesBlock)
	if err != nil {
		return err
	}
	return nil
}

//PublicKey returns the public key of local process
func (enc *Encryptor) PublicKey() []byte {
	return elliptic.Marshal((*enc).curve, (*enc).publicKeyX, (*enc).publicKeyY)
}

//Encrypt encrypts the input slice
func (enc *Encryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, (*enc).gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return (*enc).gcm.Seal(nonce, nonce, data, nil), nil
}

//Decrypt decrypts the input slice
func (enc *Encryptor) Decrypt(data []byte) ([]byte, error) {
	size := (*enc).gcm.NonceSize()
	nonce := make([]byte, size)
	nonce, cipherText := data[:size], data[size:]
	plainText, err := (*enc).gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

//EncryptPipe reads iputReader encrypts it and pipes it in outputWriter
//note that EncryptPipe stops the fllow of the programm therefore should be executed parallel
func (enc *Encryptor) EncryptPipe(inputReader *io.ReadCloser, outputWriter *io.WriteCloser) error {
	buff := make([]byte, 256)
	var err error
	for err == nil {
		_, err = (*inputReader).Read(buff)
		if err == nil {
			buff, err = enc.Encrypt(buff)
			if err != nil {
				(*outputWriter).Close()
				return err
			}
			_, err = (*outputWriter).Write(buff)
		}
	}
	(*outputWriter).Close()
	return err
}

//DecryptPipe reads iputReader decrypts it and pipes it in outputWriter
//note that DecryptPipe stops the fllow of the programm therefore should be executed parallel
func (enc *Encryptor) DecryptPipe(inputReader *io.ReadCloser, outputWriter *io.WriteCloser) error {
	buff := make([]byte, 256)
	var err error
	for err == nil {
		_, err = (*inputReader).Read(buff)
		if err == nil {
			buff, err = enc.Decrypt(buff)
			if err != nil {
				(*outputWriter).Close()
				return err
			}
			_, err = (*outputWriter).Write(buff)
		}
	}
	(*outputWriter).Close()
	return err
}
