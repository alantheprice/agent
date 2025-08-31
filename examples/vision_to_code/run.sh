#!/bin/bash

# Vision to Code Pipeline Demo
# This script converts UI screenshots/mockups to working code

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}📱 Vision to Code Pipeline${NC}"
echo "=================================="
echo

# Check if binary exists
BINARY="../../bin/agent-template"
if [ ! -f "$BINARY" ]; then
    echo -e "${YELLOW}Building agent-template binary...${NC}"
    cd ../..
    go build -o bin/agent-template .
    cd examples/vision_to_code
fi

# Check for required environment variables
if [ -z "$OPENAI_API_KEY" ] && [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$DEEPINFRA_API_KEY" ]; then
    echo -e "${RED}Error: Vision-capable LLM API key required${NC}"
    echo "Export one of these environment variables:"
    echo "  export OPENAI_API_KEY='your-key-here'     (GPT-4 Vision)"
    echo "  export ANTHROPIC_API_KEY='your-key-here'  (Claude Vision)"
    echo "  export DEEPINFRA_API_KEY='your-key-here'  (Vision models)"
    exit 1
fi

# Default values
IMAGE_PATH="${1}"
CONFIG="${2:-vision_to_code_pipeline.json}"
FRAMEWORK="${3:-react}"

# Check if image file is provided
if [ -z "$IMAGE_PATH" ]; then
    echo -e "${RED}Error: Image path required${NC}"
    echo
    echo -e "${YELLOW}Usage:${NC}"
    echo "  ./run.sh /path/to/screenshot.png"
    echo "  ./run.sh /path/to/mockup.jpg react"
    echo "  ./run.sh /path/to/design.png vision_to_code_pipeline.json vue"
    echo
    echo -e "${YELLOW}Supported frameworks:${NC} react, vue, html, flutter"
    echo -e "${YELLOW}Supported image formats:${NC} .png, .jpg, .jpeg, .webp"
    exit 1
fi

# Check if image file exists
if [ ! -f "$IMAGE_PATH" ]; then
    echo -e "${RED}Error: Image file not found: $IMAGE_PATH${NC}"
    exit 1
fi

echo -e "${BLUE}Configuration:${NC}"
echo "  Image File: $IMAGE_PATH"
echo "  Pipeline Config: $CONFIG"
echo "  Target Framework: $FRAMEWORK"
echo "  Output Directory: ./output/"
echo

# Get absolute path for image
IMAGE_ABS_PATH=$(realpath "$IMAGE_PATH")

# Create output directory
mkdir -p output

# Prepare input data with image
echo -e "${YELLOW}Starting Vision to Code conversion...${NC}"
echo "Analyzing image and generating code..."
echo

$BINARY run \
    --config "$CONFIG" \
    --input-data "{
        \"image_path\": \"$IMAGE_ABS_PATH\",
        \"framework\": \"$FRAMEWORK\",
        \"output_dir\": \"./output\",
        \"generation_type\": \"component\",
        \"include_styling\": true,
        \"responsive\": true
    }" \
    --output-dir "./output"

echo
echo -e "${GREEN}✅ Code generation completed!${NC}"
echo

# Show generated files
if [ -d "output" ]; then
    echo -e "${BLUE}Generated files:${NC}"
    find output -type f -name "*.js" -o -name "*.jsx" -o -name "*.vue" -o -name "*.html" -o -name "*.css" -o -name "*.dart" | head -10
    echo
fi

# Show sample of generated code if it exists
CODE_FILE=""
case $FRAMEWORK in
    "react")
        CODE_FILE=$(find output -name "*.jsx" -o -name "*.js" | head -1)
        ;;
    "vue")
        CODE_FILE=$(find output -name "*.vue" | head -1)
        ;;
    "html")
        CODE_FILE=$(find output -name "*.html" | head -1)
        ;;
    "flutter")
        CODE_FILE=$(find output -name "*.dart" | head -1)
        ;;
esac

if [ -n "$CODE_FILE" ] && [ -f "$CODE_FILE" ]; then
    echo -e "${BLUE}Generated Code Preview ($CODE_FILE):${NC}"
    head -30 "$CODE_FILE"
    echo "..."
    echo
fi

# Show instructions if they exist
if [ -f "output/implementation_guide.md" ]; then
    echo -e "${BLUE}Implementation Guide Preview:${NC}"
    head -20 output/implementation_guide.md
    echo "..."
    echo
fi

echo -e "${YELLOW}Usage Examples:${NC}"
echo "  ./run.sh screenshot.png"
echo "  ./run.sh ui_mockup.jpg vision_to_code_pipeline.json vue"
echo "  ./run.sh mobile_design.png vision_to_code_pipeline.json flutter"
echo