module github.com/sujal-lgtm/Contextify

go 1.23.0

toolchain go1.24.6

replace github.com/sujal-lgtm/Contextify/backend/pkg/producer => ../../pkg/producer

require github.com/lib/pq v1.10.9
