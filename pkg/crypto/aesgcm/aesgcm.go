package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func Encrypt(plainText string, encyptedKey string, userData []byte) ([]byte, error) {
	// декодирование мастер-ключа из Base64
	key, err := base64.StdEncoding.DecodeString(encyptedKey)
	if err != nil {
		return nil, fmt.Errorf("decoding master key: %w", err)
	}

	// инициализация aes
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes cipher: %w", err)
	}

	// создание обертки над aes, которая добавляет логику gcm
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init aes-gcm cipher: %w", err)
	}

	// генерация nonce - уникального числа для каждой записи - нужно для уникальности шифра
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate rand for nonce: %w", err)
	}

	// шифрование
	// в dst передаем 'nonce', чтобы функция приклеила зашифрованные данные сразу после nonce
	// в итоге получим структуру: [NONCE (12b)] + [CIPHERTEXT] + [TAG (16b)].
	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), userData)

	return cipherText, nil
}

func Decrypt(encryptedData []byte, key []byte, userData []byte) (string, error) {
	// инициализация aes
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("init aes cipher: %w", err)
	}

	// создание обертки над aes, которая добавляет логику gcm
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("init aes-gcm cipher: %w", err)
	}

	// проверка первых 12 байт, которые заложены под nonce
	nonceSize := aesGCM.NonceSize()
	if len(encryptedData) < nonceSize {
		return "", fmt.Errorf("encrypted data is too short")
	}

	// разделяем слайс
	// первые 12 байт это nonce, всё остальное - cipherText + tag
	nonce, cipherText := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// расшифровка и проверка
	plainText, err := aesGCM.Open(nil, nonce, cipherText, userData)
	if err != nil {
		return "", fmt.Errorf("decrypt cipher text: %w", err)
	}

	return string(plainText), nil
}
