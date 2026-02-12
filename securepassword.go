package hashkit

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const argon2Version = 19

// ===============================
// Config
// ===============================
type Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
	SaltLength  uint32
	Workers     int
}

func DefaultConfig() *Config {
	return &Config{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: uint8(runtime.NumCPU()),
		KeyLength:   32,
		SaltLength:  16,
		Workers:     runtime.NumCPU(),
	}
}

// ===============================
// Engine
// ===============================
type Engine struct {
	cfg      *Config
	rehashCh chan rehashJob
	wg       sync.WaitGroup
}

type rehashJob struct {
	password string
	callback func(string)
}

func NewEngine(cfg *Config) *Engine {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	e := &Engine{
		cfg:      cfg,
		rehashCh: make(chan rehashJob, cfg.Workers),
	}

	for i := 0; i < cfg.Workers; i++ {
		e.wg.Add(1)
		go e.worker()
	}

	return e
}

func (e *Engine) worker() {
	defer e.wg.Done()
	for job := range e.rehashCh {
		hash, _ := e.Hash(job.password)
		job.callback(hash)
	}
}

func (e *Engine) Shutdown() {
	close(e.rehashCh)
	e.wg.Wait()
}

// ===============================
// Hash
// ===============================
func (e *Engine) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	salt := make([]byte, e.cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		e.cfg.Iterations,
		e.cfg.Memory,
		e.cfg.Parallelism,
		e.cfg.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		e.cfg.Memory,
		e.cfg.Iterations,
		e.cfg.Parallelism,
		b64Salt,
		b64Hash,
	), nil
}

// ===============================
// Verify
// ===============================
func (e *Engine) Verify(
	ctx context.Context,
	password string,
	encodedHash string,
	autoRehash bool,
	callback func(string),
) (bool, error) {

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8

	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	computed := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	if subtle.ConstantTimeCompare(computed, expectedHash) != 1 {
		return false, nil
	}

	// Parameter upgrade check
	if autoRehash &&
		(memory != e.cfg.Memory ||
			iterations != e.cfg.Iterations ||
			parallelism != e.cfg.Parallelism) &&
		callback != nil {

		select {
		case e.rehashCh <- rehashJob{password: password, callback: callback}:
		default:
			// queue full → skip rehash to protect performance
		}
	}

	return true, nil
}
