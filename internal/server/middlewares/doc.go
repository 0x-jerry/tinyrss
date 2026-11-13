// Package middlewares holds the HTTP middleware chain shared by the API and the
// embedded SPA: request IDs, access logging, panic recovery, and token auth.
// It is independent of the server package (no imports back into it) so both
// chains can compose these handlers without an import cycle.
package middlewares
