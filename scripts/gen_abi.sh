#!/bin/bash

# Define the source (ABI) and destination (Go bindings) directories
ABI_DIR="pkg/abi"
OUTPUT_DIR="internal/bridge/abi"

# Create the output directories for L1 and L2 if they don't exist
mkdir -p "$OUTPUT_DIR/L1"
mkdir -p "$OUTPUT_DIR/L2"

# Function to recursively process all ABI JSON files
function listFiles()
{
    # $1 is the directory to scan
    # $2 is the path to the base directory (used for output path construction)
    for file in `ls $1`; do
        if [[ -d "$1/$file" ]]; then
            listFiles "$1/$file" "$2/$file"  # Recurse into subdirectories
        else
            if echo "$file" | grep -q -E '\.dbg.json$' || echo "$file" | grep -q -E '\.go$'; then
                continue  # Skip debug JSON files and Go files
            fi
            if [[ "$file" =~ (.*)\.json ]]; then
                package=${BASH_REMATCH[1]}  # Extract the base name (without .json)
                echo "Processing $file..."

                mkdir -p "$2/$package"
                
                # Use jq to extract the ABI from the JSON and pass it to abigen
                cat "$1/$file" | jq '.abi' | abigen --abi - --pkg="$package" --out="$2/$package/$package.go"

                # Check if abigen command was successful
                if [ $? -eq 0 ]; then
                    echo "Successfully generated: $2/$package/$package.go"
                else
                    echo "Error generating: $2/$package/$package.go"
                fi
            fi
        fi
    done
}

# Call the function for L1 and L2 ABI directories
echo "Generating Go bindings for L1 ABI..."
listFiles "$ABI_DIR/L1" "$OUTPUT_DIR/L1"
echo "Generating Go bindings for L2 ABI..."
listFiles "$ABI_DIR/L2" "$OUTPUT_DIR/L2"

echo "Go bindings generation completed."