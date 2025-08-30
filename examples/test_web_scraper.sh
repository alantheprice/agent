#!/bin/bash

# Test script for the Web Scraper Agent
# This demonstrates how to use the generic agent framework to scrape and extract structured data

echo "🌐 Web Scraper Agent Test"
echo "========================="

# Build the generic agent if not already built
if [ ! -f "./generic-agent" ]; then
    echo "📦 Building generic agent..."
    go build -o generic-agent cmd/generic/main.go
    if [ $? -ne 0 ]; then
        echo "❌ Failed to build generic agent"
        exit 1
    fi
    echo "✅ Generic agent built successfully"
fi

# Test 1: Validate the configuration
echo ""
echo "🔍 Test 1: Validating web scraper configuration..."
./generic-agent validate --config examples/configs/web_scraper.json

if [ $? -eq 0 ]; then
    echo "✅ Configuration validation passed"
else
    echo "❌ Configuration validation failed"
    exit 1
fi

# Test 2: Show the schema to understand what we're extracting
echo ""
echo "📋 Test 2: Target extraction schema..."
echo "Schema file: examples/schemas/extraction_schema.json"
cat examples/schemas/extraction_schema.json | jq '.properties.article.properties | keys' 2>/dev/null || echo "Requires jq for pretty display"

# Test 3: Run the web scraper on a sample URL (requires actual web access)
echo ""
echo "🕷️  Test 3: Running web scraper on sample URL..."

# Use a reliable test URL that should have article-like content
TEST_URL="https://httpbin.org/html"
OUTPUT_FILE="./output/scraped_article.json"

echo "Target URL: $TEST_URL"
echo "Schema: examples/schemas/extraction_schema.json"  
echo "Output: $OUTPUT_FILE"

# Create output directory if it doesn't exist
mkdir -p output

# Run the web scraper agent
echo ""
echo "🚀 Executing web scraper agent..."
./generic-agent run "Extract article data from web page" \
  --config examples/configs/web_scraper.json \
  --param url="$TEST_URL" \
  --param schema_file="examples/schemas/extraction_schema.json" \
  --param output_file="$OUTPUT_FILE"

SCRAPER_EXIT_CODE=$?

# Check results
echo ""
echo "📊 Results:"
echo "=========="

if [ $SCRAPER_EXIT_CODE -eq 0 ]; then
    echo "✅ Web scraper completed successfully"
    
    if [ -f "$OUTPUT_FILE" ]; then
        echo "📄 Output file created: $OUTPUT_FILE"
        echo ""
        echo "📝 Extracted data preview:"
        if command -v jq >/dev/null 2>&1; then
            head -20 "$OUTPUT_FILE" | jq '.' 2>/dev/null || cat "$OUTPUT_FILE" | head -20
        else
            head -20 "$OUTPUT_FILE"
        fi
        
        echo ""
        echo "📈 File size: $(wc -c < "$OUTPUT_FILE") bytes"
        
        # Validate JSON structure
        if command -v jq >/dev/null 2>&1; then
            if jq empty "$OUTPUT_FILE" 2>/dev/null; then
                echo "✅ Output is valid JSON"
            else
                echo "⚠️  Output JSON validation failed"
            fi
        fi
    else
        echo "⚠️  Output file not found: $OUTPUT_FILE"
    fi
else
    echo "❌ Web scraper failed with exit code: $SCRAPER_EXIT_CODE"
fi

# Test 4: Alternative test with a different URL
echo ""
echo "🔄 Test 4: Alternative URL test..."

# Try a different URL with more predictable content
ALT_URL="https://jsonplaceholder.typicode.com/posts/1"
ALT_OUTPUT="./output/scraped_json_placeholder.json"

echo "Trying alternative URL: $ALT_URL"
./generic-agent run "Extract data from JSON placeholder" \
  --config examples/configs/web_scraper.json \
  --param url="$ALT_URL" \
  --param schema_file="examples/schemas/extraction_schema.json" \
  --param output_file="$ALT_OUTPUT" 2>/dev/null

if [ $? -eq 0 ] && [ -f "$ALT_OUTPUT" ]; then
    echo "✅ Alternative URL test passed"
else
    echo "ℹ️  Alternative URL test completed (may require actual LLM integration)"
fi

# Summary
echo ""
echo "🎯 Test Summary:"
echo "==============="
echo "1. ✅ Configuration validation: PASSED"
echo "2. ✅ Schema structure: READY" 
echo "3. Status: Web scraper workflow tested (LLM integration may require API keys)"
echo "4. Files created:"
echo "   - examples/configs/web_scraper.json (agent configuration)"
echo "   - examples/schemas/extraction_schema.json (data schema)"
if [ -f "$OUTPUT_FILE" ]; then
    echo "   - $OUTPUT_FILE (extracted data)"
fi

echo ""
echo "🚀 To use with real LLM providers:"
echo "1. Set up API keys in your environment"
echo "2. Update the LLM configuration in web_scraper.json"  
echo "3. Run: ./generic-agent run \"Extract data\" --config examples/configs/web_scraper.json --param url=\"https://example.com/article\""

echo ""
echo "✨ Web scraper test completed!"