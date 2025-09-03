# wwiseutil-SDDE

[Read in Chinese (中文说明)](README.zh-CN.md)

This is a command-line tool for Wwise audio packages (`.pck` and `.bnk` files), specifically customized to provide **file replacement** and **repacking** functionality for the `.pck` file format used in **Sleeping Dogs: Definitive Edition**.

This project is a fork and improvement of [hpxro7/wwiseutil](https://github.com/hpxro7/wwiseutil). Special thanks to the original author for their work.

Modified and maintained by [Zinho](https://github.com/ZinhoYip).

## Core Features

- **Unpack Files**: Extract all embedded sub-files from `.pck` and `.bnk` archives.
- **Inspect File Structure**: Clearly list the IDs and indices of all `.bnk` and `.wem` files within a `.pck` package using the verbose log mode.
- **Replace and Repack (SDDE Custom)**: Replace specific files within a `.pck` package and `.bnk` package with new ones and generate a new, game-ready `.pck` file and `.bnk` file.
- **Cross-Platform**: Built with Go, it can be easily compiled and run on Windows, macOS, and Linux.

## How to Use `wwiseutil_SDDE.exe`

You can download the latest version of `wwiseutil_SDDE.exe` from the [GitHub Releases](https://github.com/ZinhoYip/wwiseutil-SDDE/releases) page of this project.

After downloading, it is recommended to place it in an easily accessible path or add it to your system's environment variables for convenient access from any command-line location.

### 1. Inspect `.pck` Package Contents (Preparation for File Replacement)

Before replacing files, you **must** know the index of each file inside the package. Using the `-v` (verbose) parameter generates a `log.txt` file containing detailed information about all files.

**Command Format:**
```bash
wwiseutil_SDDE.exe -f "<path_to_source.pck>" -v -u -o "<any_temporary_output_dir>"
```

**Example:**
```bash
# This will generate a log.txt file in the same directory as the program
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -v -u -o "F:\temp_unpack_output"
```

Open `log.txt`, and you will see content similar to the following. Take note of the **Index** of the file you want to replace.

### 2. Unpack `.pck` or `.bnk` Files

If you just want to extract all files from a package, use the `-u` (unpack) parameter.

**Command Format:**
```bash
wwiseutil_SDDE.exe -f "<path_to_source.pck_or.bnk>" -u -o "<output_directory>"
```

**Example:**
```bash
# Unpack a PCK file
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -u -o "C:\unpacked_pck_files"

# Unpack a BNK file
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\135561656.bnk" -u -o "C:\unpacked_bnk_files"
```

### 3. Replace Files in a `.pck` (Core Feature)

This is the core feature customized for SDDE. Please follow these steps strictly.

**Step 1: Prepare Replacement Files**

1.  Create a main folder to hold all your replacement files (e.g., `D:\my_replacements`).
2.  Inside this main folder, create `bnk` and/or `wem` subfolders, depending on the type of file you are replacing.
3.  Place the `.bnk` or `.wem` files you want to use for replacement into the corresponding subfolders.
4.  **Crucially:** Rename these files to the **Index number** you found in Step 1.
    -   For example, to replace `BnkIndex[1]`, rename your new bnk file to `1.bnk` and place it in the `bnk` folder.
    -   To replace `WemIndex[5]`, rename your new wem file to `5.wem` and place it in the `wem` folder.

**Directory Structure Example:**
```
D:\my_replacements\
├───bnk\
│   └───1.bnk
└───wem\
    └───5.wem
```

**Step 2: Execute the Replace Command**

Use the `-r` (replace) parameter, providing the source file, the replacement directory, and the path for the new output file.

**Command Format:**
```bash
wwiseutil_SDDE.exe -f "<path_to_source.pck>" -r -t "<your_main_replacement_dir>" -o "<path_for_newly_generated.pck>"
```

**Example:**
```bash
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -r -t "D:\my_replacements" -o "C:\SDDE\Data\Audio\SD2\sfx_new.pck"
```

After the command completes successfully, `sfx_new.pck` is the new file containing your modified content. You can rename it back to `sfx.pck` and replace the original game file to test it.

## Acknowledgments

-   Thanks to **hpxro7** for creating the original [wwiseutil](https://github.com/hpxro7/wwiseutil).

## License

This project is licensed under the GNU General Public License v3.0. See the `LICENSE` file for details.
