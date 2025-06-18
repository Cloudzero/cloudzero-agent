// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package monitor provides functionality to manage and reload TLS certificates dynamically.
// This package is designed to enhance the security and flexibility of TLS configurations by
// allowing dynamic reloading and rotation of TLS certificates. It supports various methods
// for triggering certificate reloads, such as file system changes, SIGHUP signals, and
// periodic intervals.
//
// The main component of this package is the reconciler struct, which manages the TLS
// configuration and ensures that the latest certificates are always used. The reconciler
// can be customized using functional options to fit different use cases.
//
// This package is valuable for applications that require robust and flexible TLS certificate
// management, ensuring that the TLS configuration stays up-to-date with the latest certificates
// without requiring application restarts.
package monitor

import (
	"container/list"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	// The singleflight package is part of the extended Go libraries (golang.org/x/)
	// and is designed to address the "duplicate suppression" problem.
	// When multiple goroutines need the same data or perform the same computation concurrently,
	// singleflight ensures that only one of them executes the operation, while the others wait for the result.
	"golang.org/x/sync/singleflight"
)

// TLSProvider defines an interface for retrieving TLS certificates.
// Implementing this interface allows for dynamic reloading and rotation of TLS certificates,
// enhancing the security and flexibility of TLS configurations.
type TLSProvider interface {
	// Certificates retrieves the most recently rotated client or server (leaf) certificate
	// along with the root CA certificates.
	// It returns an error if the retrieval fails.
	//
	// Note:
	// - The method may return nil for root CAs if a predefined CA pool is set in tls.Config.
	// - This allows for flexibility in managing CA pools, either dynamically or statically.
	Certificates() (*tls.Certificate, []*x509.Certificate, error)
}

// Option represents a configuration option for the reconciler.
// It follows the functional options pattern, allowing for flexible and readable configuration.
type Option interface {
	apply(*reconciler)
}

// optionFunc is a function type that implements the Option interface.
// It allows regular functions to be used as configuration options.
type optionFunc func(*reconciler)

// apply executes the configuration option on the provided reconciler.
func (fn optionFunc) apply(r *reconciler) {
	fn(r)
}

// WithVerifyConnection configures the TLS settings to verify the TLS connection's certificate.
// It sets tls.Config.VerifyConnection to use the reconciler's CA pool, which is managed based on certificate rotations.
// If no custom CA pool is provided, the system roots or platform verifier are used.
//
// Additionally, this option sets tls.Config.InsecureSkipVerify to true to bypass the default Go TLS validation.
// This does not disable VerifyConnection but allows the reconciler to handle certificate verification.
//
// Note:
// - The tls.Config.RootCAs and tls.Config.ClientCAs are ignored when this option is applied.
// - This option should be used in conjunction with WithRootsLimit for effective certificate pool management.
//
// Example:
//
//	tlsConfig := TLSConfig(WithVerifyConnection())
func WithVerifyConnection() Option {
	return optionFunc(func(r *reconciler) {
		// Set the custom VerifyConnection function provided by the reconciler.
		r.config.VerifyConnection = r.verifyConnection
		// Instruct tls.Config to skip the default verification to allow the custom VerifyConnection to take over.
		r.config.InsecureSkipVerify = true
	})
}

// WithProvider sets the TLSProvider for the reconciler.
// The provider is responsible for retrieving the latest certificates upon receiving a reload signal.
//
// This option configures the tls.Config to use the reconciler's methods for obtaining certificates,
// ensuring that the TLS configuration stays up-to-date with the latest certificates.
//
// Example:
//
//	provider := NewMyTLSProvider()
//	tlsConfig := TLSConfig(WithProvider(provider))
func WithProvider(p TLSProvider) Option {
	return optionFunc(func(r *reconciler) {
		// Assign the provided TLSProvider to the reconciler.
		r.p = p
		// Set the GetCertificate and GetClientCertificate callbacks to the reconciler's methods.
		r.config.GetCertificate = r.getCertificate
		r.config.GetClientCertificate = r.getClientCertificate
	})
}

// WithCertificatesPaths configures the reconciler to load TLS certificates from specified file paths.
// It sets up a file system provider that watches the provided certificate and key files,
// and optionally a CA bundle for root CAs.
//
// Parameters:
// - cert: Path to the TLS certificate file.
// - key: Path to the TLS key file.
// - ca: Path to the CA bundle file. Can be empty if no CA rotation is needed.
//
// Note:
// - The CA path can be empty if root CA rotation is not required.
// - If a CA path is provided, it should point to a valid CA bundle.
//
// Example:
//
//	tlsConfig := TLSConfig(WithCertificatesPaths("/path/to/cert.pem", "/path/to/key.pem", "/path/to/ca.pem"))
func WithCertificatesPaths(cert, key, ca string) Option {
	// Create a fileSystemProvider with the provided paths and pass it to WithProvider.
	return WithProvider(fileSystemProvider{ca, cert, key})
}

// WithRootsLimit sets a limit on the number of old root CA certificates to retain in the pool.
// This is useful for maintaining backward compatibility, allowing verification of certificates
// issued before recent rotations.
//
// Parameters:
// - n: The maximum number of root CA certificates to keep.
//
// Note:
//   - This option works in tandem with WithVerifyConnection. If WithVerifyConnection is not used,
//     this option has no effect.
//
// Default:
// - If not set, the default limit is 2.
//
// Example:
//
//	tlsConfig := TLSConfig(WithRootsLimit(3))
func WithRootsLimit(n uint) Option {
	return optionFunc(func(r *reconciler) {
		// Set the rootsLimit in the reconciler.
		r.rootsLimit = n
	})
}

// WithReloadFunc registers a custom function to determine when a certificate reload is needed.
// This function is periodically or conditionally called to check if a reload should occur.
//
// Parameters:
// - f: A function that returns a boolean indicating whether a reload is necessary.
//
// Note:
// - Multiple goroutines may invoke this function concurrently.
// - Ensure that the function is thread-safe.
//
// Example:
//
//	reloadFunc := func() bool {
//	    // Custom logic to determine if reload is needed
//	    return time.Now().Unix()%60 == 0
//	}
//	tlsConfig := TLSConfig(WithReloadFunc(reloadFunc))
func WithReloadFunc(f func() bool) Option {
	return optionFunc(func(r *reconciler) {
		// Assign the custom reload function to the reconciler.
		r.reload = f
	})
}

// WithSIGHUPReload configures the reconciler to reload certificates upon receiving a SIGHUP signal.
// This is useful for triggering certificate reloads without restarting the application.
//
// Parameters:
// - c: A channel that receives os.Signal notifications.
//
// Example:
//
//	signalChan := make(chan os.Signal, 1)
//	signal.Notify(signalChan, syscall.SIGHUP)
//	tlsConfig := TLSConfig(WithSIGHUPReload(signalChan))
func WithSIGHUPReload(c chan os.Signal) Option {
	return optionFunc(func(r *reconciler) {
		// Define the reload function to listen for SIGHUP signals.
		r.reload = func() bool {
			select {
			case sig := <-c:
				// Trigger reload only if the received signal is SIGHUP.
				if sig == syscall.SIGHUP {
					return true
				}
				return false
			default:
				// No signal received; do not reload.
				return false
			}
		}
	})
}

// WithDurationReload configures the reconciler to reload certificates at specified intervals.
// This is useful for ensuring that certificates are periodically refreshed.
//
// Parameters:
// - dur: The duration between each reload attempt.
//
// Example:
//
//	tlsConfig := TLSConfig(WithDurationReload(24*time.Hour))
func WithDurationReload(dur time.Duration) Option {
	return optionFunc(func(r *reconciler) {
		// Create a mutex to synchronize access to the next reload time.
		mu := new(sync.Mutex)
		// Initialize the next reload time based on the current time plus the duration.
		nextReload := time.Now().Add(dur)

		// Define the reload function to check if the specified duration has elapsed.
		r.reload = func() bool {
			mu.Lock()
			defer mu.Unlock()

			// If the current time has passed the next reload time, schedule the next reload and trigger a reload.
			if time.Now().After(nextReload) {
				nextReload = time.Now().Add(dur)
				return true
			}

			// Otherwise, do not reload.
			return false
		}
	})
}

// WithOnReload registers a callback function to be invoked after a certificate reload.
// This can be used for additional actions such as rotating session tickets or logging.
//
// Parameters:
// - f: A function that takes a pointer to tls.Config and performs custom actions.
//
// Note:
// - The callback is executed in its own goroutine to avoid blocking the reconciler.
//
// Example:
//
//	onReload := func(cfg *tls.Config) {
//	    log.Println("TLS certificates reloaded")
//	}
//	tlsConfig := TLSConfig(WithOnReload(onReload))
func WithOnReload(f func(*tls.Config)) Option {
	return optionFunc(func(r *reconciler) {
		// Assign the callback function to be invoked upon reload.
		r.onReload = f
	})
}

// TLSConfig creates a new tls.Config that automatically reconciles certificates after rotations.
// It accepts a variadic number of Option interfaces to customize the TLS configuration.
//
// If no options are provided, it returns a default tls.Config.
//
// Example:
//
//	tlsConfig := TLSConfig(
//	    WithVerifyConnection(),
//	    WithCertificatesPaths("/path/to/cert.pem", "/path/to/key.pem", "/path/to/ca.pem"),
//	    WithRootsLimit(3),
//	    WithReloadFunc(myReloadFunc),
//	)
//
//	server := &http.Server{
//	    Addr:      ":443",
//	    TLSConfig: tlsConfig,
//	    // Other server settings...
//	}
func TLSConfig(opts ...Option) *tls.Config {
	// If no options are provided, return a default tls.Config.
	if len(opts) == 0 {
		return new(tls.Config)
	}

	// Initialize a new reconciler with default settings.
	reconciler := newReconciler()

	// Apply each provided option to the reconciler.
	for _, opt := range opts {
		opt.apply(reconciler)
	}

	// Return the configured tls.Config from the reconciler.
	return reconciler.config
}

// newReconciler initializes a new reconciler with default settings.
// It sets up the singleflight.Group and other necessary fields for certificate management.
func newReconciler() *reconciler {
	return &reconciler{
		// Default limit for root CAs to maintain backward compatibility.
		rootsLimit: 2,
		// Initialize singleflight.Group to prevent duplicate reloads.
		flight: &singleflight.Group{},
		// Initialize condition variable with a no-operation locker.
		cond: sync.NewCond(noopLocker{}),
		// Assign a no-operation TLSProvider by default.
		p: noopProvider{},
		// Initialize a list to manage root CAs.
		ll: list.New(),
		// Create a new tls.Config to be managed by the reconciler.
		config: new(tls.Config),
		// Default reload function does not trigger a reload.
		reload: func() bool { return false },
	}
}

// reconciler manages TLS certificate reloading and rotation.
// It ensures that certificate updates are handled safely and efficiently,
// preventing redundant reloads and maintaining a pool of valid root CAs.
type reconciler struct {
	// reloading indicates whether a certificate reload is currently in progress.
	// It is used in the hot path to quickly check the reload status.
	// Placing it first in the struct can optimize memory layout on some architectures.
	reloading uint32

	// rootsLimit sets the maximum number of old root CA certificates to retain.
	// This helps in maintaining backward compatibility by allowing verification of older certificates.
	rootsLimit uint

	// flight ensures that each reload operation is executed only once,
	// regardless of the number of concurrent callers requesting a reload.
	flight *singleflight.Group

	// once ensures that the initial reload is triggered only once.
	once sync.Once

	// cond is a condition variable used to synchronize goroutines during reloads.
	cond *sync.Cond

	// pool holds the current pool of root CA certificates.
	// It is accessed atomically to ensure thread-safe reads and writes.
	pool atomic.Value

	// cert holds the current TLS certificate.
	// It is accessed atomically to ensure thread-safe reads and writes.
	cert atomic.Value

	// p represents the TLSProvider responsible for supplying certificates.
	p TLSProvider

	// ll maintains a list of root CA certificates, limited by rootsLimit.
	// It stores both new and old root CAs to manage the pool effectively.
	ll *list.List

	// config is the tls.Config that the reconciler modifies to manage TLS settings.
	config *tls.Config

	// reload is a function that determines whether a certificate reload is needed.
	// It can be customized using WithReloadFunc, WithSIGHUPReload, or WithDurationReload.
	reload func() bool

	// onReload is a callback function invoked after a successful certificate reload.
	// It can be used for additional actions like rotating session tickets or logging.
	onReload func(*tls.Config)
}

// getCertificate retrieves the current TLS certificate for server-side TLS configurations.
// It implements tls.Config.GetCertificate and ensures that the latest certificate is used.
//
// Parameters:
// - clientHello: Information about the TLS client hello.
//
// Returns:
// - The current tls.Certificate.
// - An error if the certificate retrieval fails.
func (r *reconciler) getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Retrieve the current certificate and handle any errors.
	cert, _, err := r.certificates()
	return cert, err
}

// getClientCertificate retrieves the current TLS certificate for client-side TLS configurations.
// It implements tls.Config.GetClientCertificate and ensures that the latest certificate is used.
//
// Parameters:
// - certificateRequest: Information about the TLS certificate request.
//
// Returns:
// - The current tls.Certificate.
// - An error if the certificate retrieval fails.
func (r *reconciler) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	// Retrieve the current certificate and handle any errors.
	cert, _, err := r.certificates()
	return cert, err
}

// verifyConnection validates the TLS connection's certificate using the latest root CA pool.
// It implements tls.Config.VerifyConnection, providing custom certificate verification logic.
//
// Parameters:
// - cs: The TLS connection state containing peer certificates.
//
// Returns:
// - An error if the verification fails.
func (r *reconciler) verifyConnection(cs tls.ConnectionState) error {
	// Retrieve the current root CA pool and handle any errors.
	_, pool, err := r.certificates()
	if err != nil {
		return err
	}

	// Skip verification if client certificates are not required and none are provided.
	if r.config.ClientAuth < tls.VerifyClientCertIfGiven &&
		len(cs.PeerCertificates) == 0 {
		return nil
	}

	// Set up verification options.
	opts := x509.VerifyOptions{
		Roots:         pool,                // Use the latest root CAs.
		DNSName:       r.config.ServerName, // Verify the server name.
		Intermediates: x509.NewCertPool(),  // Pool for intermediate certificates.
	}

	// Use the custom time function if provided.
	if r.config.Time != nil {
		opts.CurrentTime = r.config.Time()
	}

	// Specify key usages based on client authentication settings.
	if r.config.ClientAuth >= tls.VerifyClientCertIfGiven {
		opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// Add intermediate certificates to the verification pool.
	// The first certificate is the leaf certificate, so skip it.
	for _, inter := range cs.PeerCertificates[1:] {
		opts.Intermediates.AddCert(inter)
	}

	// Perform the certificate verification.
	_, err = cs.PeerCertificates[0].Verify(opts)
	return err
}

// certificates retrieves the current TLS certificate and root CA pool.
// It handles certificate reloads if necessary and ensures thread-safe access to certificates.
//
// Returns:
// - The current tls.Certificate.
// - The current root CA pool.
// - An error if certificate retrieval fails.
func (r *reconciler) certificates() (cert *tls.Certificate, pool *x509.CertPool, err error) {
	// Wait if a reload is currently in progress to prevent race conditions.
	for atomic.LoadUint32(&r.reloading) == 1 {
		r.cond.Wait()
	}

	// Check if a reload is needed based on the reload function.
	if r.needReload() {
		// Indicate that a reload is in progress.
		atomic.StoreUint32(&r.reloading, 1)

		// Use singleflight to ensure only one reload operation occurs,
		// even if multiple goroutines request a reload simultaneously.
		_, err, _ = r.flight.Do("reconciler", func() (interface{}, error) {
			// Retrieve certificates from the provider.
			latestCert, roots, certErr := r.p.Certificates()
			if certErr != nil {
				return nil, certErr
			}

			// If root CAs are provided, update the CA pool.
			if len(roots) > 0 {
				poolToUpdate := x509.NewCertPool()

				// Add new root CAs to the front of the list.
				for _, ca := range roots {
					_ = r.ll.PushFront(ca)
					// Remove the oldest root CA if the limit is exceeded.
					if uint(r.ll.Len()) > r.rootsLimit { //nolint:gosec // the length of the linked list will never be negative
						e := r.ll.Back()
						_ = r.ll.Remove(e)
					}
				}

				// Rebuild the CA pool from the list of root CAs.
				for e := r.ll.Front(); e != nil; e = e.Next() {
					if cert, ok := e.Value.(*x509.Certificate); ok {
						poolToUpdate.AddCert(cert)
					}
				}

				// Atomically store the updated CA pool.
				r.pool.Store(poolToUpdate)
			}

			// Atomically store the updated TLS certificate.
			r.cert.Store(latestCert)

			// Invoke the onReload callback if it is set.
			if r.onReload != nil {
				go r.onReload(r.config)
			}

			// Indicate that the reload has completed.
			atomic.StoreUint32(&r.reloading, 0)
			// Broadcast to all waiting goroutines that the reload is done.
			r.cond.Broadcast()
			return nil, nil //nolint:nilnil // we don't actually use interface return value, but it's part of the singleflight API
		})
	}

	// Load the current certificate atomically.
	if v, ok := r.cert.Load().(*tls.Certificate); ok {
		cert = v
	}

	// Load the current root CA pool atomically.
	if v, ok := r.pool.Load().(*x509.CertPool); ok {
		pool = v
	}

	return cert, pool, err
}

// needReload determines whether a certificate reload is necessary.
// It triggers the initial reload on the first call and subsequently relies on the custom reload function.
func (r *reconciler) needReload() (ok bool) {
	// Ensure that the initial reload is triggered only once.
	r.once.Do(func() {
		ok = true
	})

	// Return true if it's the first call or if the custom reload function indicates a reload is needed.
	return ok || r.reload()
}

// fileSystemProvider is a simple implementation of TLSProvider that loads certificates from the file system.
// It expects a slice of strings containing paths to the CA bundle, certificate, and key files, respectively.
type fileSystemProvider []string

// Certificates loads TLS certificates from the file system.
// It expects exactly three paths: CA bundle (can be empty), certificate, and key.
//
// Returns:
// - The loaded tls.Certificate.
// - A slice of root CA certificates.
// - An error if loading fails.
//
// Example:
//
//	provider := fileSystemProvider{"/path/to/ca.pem", "/path/to/cert.pem", "/path/to/key.pem"}
//	cert, roots, err := provider.Certificates()
func (fsp fileSystemProvider) Certificates() (*tls.Certificate, []*x509.Certificate, error) {
	// Ensure that exactly three paths are provided.
	if len(fsp) != 3 {
		return nil, nil, errors.New("tlsreconciler: certificates path missing")
	}

	caFile, certFile, keyFile := fsp[0], fsp[1], fsp[2]

	// Load the TLS certificate and key from the specified files.
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}

	// If no CA file is provided, return the certificate with nil roots.
	if len(caFile) == 0 {
		return &cert, nil, nil
	}

	// Read the CA bundle file.
	caPEMBlock, err := os.ReadFile(caFile)
	if err != nil {
		return nil, nil, err
	}

	var (
		p     *pem.Block
		roots []*x509.Certificate
	)

	// Decode and parse each PEM block in the CA bundle.
	for {
		p, caPEMBlock = pem.Decode(caPEMBlock)
		if p == nil {
			break
		}

		// Parse the certificate from the PEM block.
		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			return nil, nil, err
		}

		// Append the parsed certificate to the roots slice.
		roots = append(roots, cert)
	}

	return &cert, roots, nil
}

// noopLocker is a no-operation implementation of sync.Locker.
// It is used to satisfy the sync.Cond requirement without actual locking,
// effectively making the condition variable always ready.
type noopLocker struct{}

// Lock is a no-operation method to satisfy sync.Locker interface.
func (noopLocker) Lock() {}

// Unlock is a no-operation method to satisfy sync.Locker interface.
func (noopLocker) Unlock() {}

// noopProvider is a no-operation implementation of TLSProvider.
// It always returns nil for certificates and roots, effectively disabling certificate loading.
type noopProvider struct{}

// Certificates returns nil for both tls.Certificate and root CAs, with no error.
// It serves as a default provider when no other provider is specified.
func (noopProvider) Certificates() (*tls.Certificate, []*x509.Certificate, error) {
	return nil, nil, nil
}
