#!/bin/bash
# Script to generate MD5 checksums for files in the current directory
# Author: ChatGPT

# Array of file extensions to ignore
ignores=("sh" "blend")
result=""

# Iterate through the ignore list to build the regex patterns for the find command
for filetype in "${ignores[@]}"; do
  # If result is empty, start building the regex pattern
  if [ -z "$result" ]; then
    result="-regex .*\\.$filetype"
  else
    # If result is not empty, append additional regex patterns using -o (logical OR)
    result+=" -o -regex .*\\.$filetype"
  fi
done

# Delete md5.txt if it exists to start fresh
[ -f md5.txt ] && rm md5.txt

# Define a function that calculates the MD5 sum for a file
checksum() {
  file="$1"
  # Remove the leading './' from the file path using parameter expansion
  trimmed_file="${file#./}"
  
  # Calculate the MD5 sum using md5sum and process the output with awk
  md5sum=$(md5sum "$file" | awk -v path="$trimmed_file" '{printf "%s %s\n", $1, path}')
  # ^-v path="$trimmed_file"  : Pass the trimmed_file variable to awk as 'path'
  # ^'{printf "%s %s\n", $1, path}' : Print the first field (MD5 sum) followed by the trimmed file path, separated by a space
  
  echo "$md5sum"
}

export -f checksum

# Get the number of available cores
num_cores=$(nproc)

# Use find command to search for files and pass them to xargs
# xargs runs the checksum function in parallel processes
find . -type d -name .git -prune -o ! \( $result \) -type f -print0 \
  | xargs -0 -I {} -P "$num_cores" bash -c 'checksum "$1" >> md5.txt' _ {}

# find command parameters explanation:
# -type d -name .git -prune : Exclude .git directories from search results
# -o                        : OR operator, used to combine multiple expressions
# ! \( $result \)           : Negate the result regex patterns (file extensions to ignore)
# -type f                   : Only search for files, not directories
# -print0                   : Print results separated by a null character (useful when file names contain spaces or special characters)

# xargs command parameters explanation:
# -0                        : Use null characters as input item separators, matching -print0 in find command
# -I {}                     : Replace occurrences of {} in the command line with the input item
# -P "$num_cores"           : Run processes in parallel, up to the number of available cores
# bash -c '...' _ {}        : Run the checksum function for each input item (file) and append the result to md5.txt
