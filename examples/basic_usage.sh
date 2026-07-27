#!/bin/bash

# === Basic usage ===

# Compress and subtitle all videos (default: 720p, 24fps, English)
./vcx.sh

# === Compression only ===

# Compress all videos to 1080p at 30fps
./vcx.sh --compress 1920x1080 --fps 30 --no-sub

# Compress without subtitles and replace originals
./vcx.sh --compress 1280x720 --no-sub --remove-original

# === Subtitles only ===

# Generate English subtitles for all videos (no compression)
./vcx.sh --no-compress

# Generate Spanish subtitles
./vcx.sh --no-compress --es

# Generate French subtitles
./vcx.sh --no-compress --fr

# === Remove original files ===

# Process all videos and replace originals
./vcx.sh --compress 1280x720 --remove-original

# Preview what would be replaced (safe)
./vcx.sh --remove-original --dry-run

# === Logging and debugging ===

# Run with verbose output
./vcx.sh --verbose

# Run quietly
./vcx.sh --quiet

# Save log to a specific file
./vcx.sh --log-file my_session.log

# Disable logging
./vcx.sh --no-log

# === Custom configuration ===

# Use a custom config file
./vcx.sh --config /path/to/config

# Example ~/.vcxrc configuration file:
#   RESOLUTION="1920x1080"
#   FPS="30"
#   LANG_CODE="en"
#   DO_COMPRESS=true
#   DO_SUBTITLE=true
#   REMOVE_ORIGINAL=false
