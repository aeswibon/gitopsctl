#!/bin/bash
# A quick script to setup standard labels for the repository using the GitHub CLI (gh)
# Ensure you are authenticated with `gh auth login` before running this.

set -e

echo "Setting up GitHub labels..."

gh label create "good first issue" --color "7057ff" --description "Good for newcomers" --force
gh label create "help wanted" --color "008672" --description "Extra attention is needed" --force
gh label create "bug" --color "d73a4a" --description "Something isn't working" --force
gh label create "enhancement" --color "a2eeef" --description "New feature or request" --force
gh label create "documentation" --color "0075ca" --description "Improvements or additions to documentation" --force

echo "Labels created successfully!"
