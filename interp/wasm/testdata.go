package wasm

// Simple WASM module that implements a solve function
// This module exports a solve function (func $solve (param i32 i32) (result i32 i32))
// It simply returns the input parameters as-is
var testWASMModule = []byte{
	0x00, 0x61, 0x73, 0x6d, // WASM_BINARY_MAGIC
	0x01, 0x00, 0x00, 0x00, // WASM_BINARY_VERSION
	// Type section
	0x01, 0x08, // section id, section size (8 bytes)
	0x01,                                     // number of types
	0x60, 0x02, 0x7f, 0x7f, 0x02, 0x7f, 0x7f, // (func (param i32 i32) (result i32 i32))
	// Function section
	0x03, 0x02, // section id, section size
	0x01, // number of functions
	0x00, // function 0, type 0
	// Memory section
	0x05, 0x03, // section id, section size
	0x01,       // number of memories
	0x00, 0x01, // memory 0: min=1 page
	// Export section
	0x07, 0x12, // section id, section size (18 bytes)
	0x02,                                                 // number of exports
	0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00, // export "memory"
	0x05, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x00, 0x00, // export "solve"
	// Code section
	0x0a, 0x08, // section id, section size (8 bytes)
	0x01,       // number of functions
	0x06,       // function body size (6 bytes)
	0x00,       // number of local declarations
	0x20, 0x00, // local.get 0
	0x20, 0x01, // local.get 1
	0x0b, // end
}

// GetTestModule returns a simple WASM module for testing
func GetTestModule() []byte {
	return testWASMModule
}
