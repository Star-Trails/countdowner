default:
    @just --list

# These tests use NSH's headless engine and need no desktop libraries.
check:
    cargo test --locked --lib

build:
    cargo build --locked --release --features native

run:
    cargo run --locked --features native
