#!/bin/bash

# Set the source and destination directories
PROTO_DIR="pkg/idl/protos"
OUTPUT_DIR="pkg/idl/pb"

# Create the output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Find all .proto files and generate Go code
find "$PROTO_DIR" -name "*.proto" | while read proto_file; do
    # Get the relative path of the proto file
    rel_path=${proto_file#"$PROTO_DIR/"}
    
    # Create the corresponding output directory
    out_dir="$OUTPUT_DIR/$(dirname "$rel_path")"
    mkdir -p "$out_dir"
    
    # Generate Go code
    protoc --go_out="$OUTPUT_DIR" --go_opt=paths=source_relative \
           --proto_path="$PROTO_DIR" "$proto_file"
    
    echo "Generated Go file for $proto_file"
done

echo "Proto generation complete!"