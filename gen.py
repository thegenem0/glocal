#!/usr/bin/env python3

import os
import argparse


def is_relevant_file(filename, file_ext):
    """
    Determine if a file is relevant (e.g., Kotlin source files).
    Modify this function to include or exclude specific files.
    """
    return filename.endswith(file_ext)


def collect_files(root_dir, file_ext):
    """
    Recursively collect all relevant files from the root directory.
    """
    file_list = []
    for dirpath, dirnames, filenames in os.walk(root_dir):
        # Exclude certain directories
        dirnames[:] = [
            d for d in dirnames if d not in ["build", ".git", ".idea", "out"]
        ]
        for filename in filenames:
            if is_relevant_file(filename, file_ext):
                full_path = os.path.join(dirpath, filename)
                file_list.append(full_path)
    return file_list


def concatenate_files(file_list, output_file):
    """
    Concatenate the contents of all files in file_list into output_file.
    """
    with open(output_file, "w", encoding="utf-8") as outfile:
        for filepath in file_list:
            with open(filepath, "r", encoding="utf-8") as infile:
                # Optionally include the file path as a comment or separator
                outfile.write(f"// File: {os.path.relpath(filepath)}\n")
                outfile.write(infile.read())
                outfile.write("\n\n")  # Add spacing between files


if __name__ == "__main__":
    DEFAULT_ROOT_DIR = "."
    DEFAULT_OUT_FILE = "context"

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--file_ext", help="extension of files to collect", required=True
    )
    parser.add_argument(
        "--root_dir",
        help="path from which to collect files",
        default=DEFAULT_ROOT_DIR,
        required=False,
    )
    parser.add_argument(
        "--out_file",
        help="name of output txt file",
        default=DEFAULT_OUT_FILE,
        required=False,
    )

    args = parser.parse_args()

    root_dir = args.root_dir
    output_file = f"{args.out_file}.txt"
    files = collect_files(root_dir, args.file_ext)
    concatenate_files(files, output_file)
    print(f"Collected {len(files)} files into {output_file}")
