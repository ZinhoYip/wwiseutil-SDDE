# wwiseutil-SDDE

这是一个针对 Wwise 音频包（`.pck` 和 `.bnk` 文件）的命令行工具，特别为游戏 **《热血无赖：终极版》（Sleeping Dogs: Definitive Edition）** 的 `.pck` 文件格式提供了定制化的**文件替换**和**重新打包**功能。

本项目并改进自 [hpxro7/wwiseutil](https://github.com/hpxro7/wwiseutil)。特别感谢原作者的工作。

由 [Zinho](https://github.com/ZinhoYip) 进行修改和维护。

## 核心功能

- **解包文件**: 从 `.pck` 和 `.bnk` 文件中提取所有包含的子文件。
- **查看文件结构**: 通过详细日志模式，清晰地列出 `.pck` 包内所有 `.bnk` 和 `.wem` 文件的ID和索引号。
- **替换并重新打包 (SDDE 定制)**: 使用新文件替换 `.pck` 包内的指定文件，并生成一个可用于游戏的新 `.pck` 文件。
- **跨平台**: 基于 Go 语言开发，可轻松编译运行于 Windows, macOS 和 Linux。

## 如何使用 `wwiseutil_SDDE.exe`

你可以从本项目的 [GitHub Releases](https://github.com/ZinhoYip/wwiseutil-SDDE/releases) 页面下载最新版本的 `wwiseutil_SDDE.exe` 可执行文件。

下载后，建议将其放置于一个方便访问的路径，或者添加到系统环境变量中，以便在任何位置的命令行中调用。

### 1. 查看 `.pck` 包内容（替换文件前的准备工作）

在替换文件之前，你**必须**知道包内每个文件的索引号（Index）。使用 `-v` (verbose) 参数可以生成一个 `log.txt` 文件，其中包含了所有文件的详细信息。

**指令格式:** 
```bash
wwiseutil_SDDE.exe -f "<源pck文件路径>" -v -u -o "<任意临时输出目录>"
```

**示例:** 
```bash
# 这会在程序同目录下生成一个 log.txt 文件
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -v -u -o "F:\temp_unpack_output"
```

打开 `log.txt`，你会看到类似下面的内容，记录好你想要替换的文件的 **Index**



 ###  2.解包 `.pck` 或 `.bnk` 文件

如果你只是想把包内的所有文件解压出来，可以使用 `-u` (unpack) 参数。

**指令格式:** 

```bash
wwiseutil_SDDE.exe -f "<源pck文件路径>" -u -o "<文件输出目录>"
```

**示例:** 
```bash
# 解包 pck 文件
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -u -o "C:\unpacked_pck_files"

# 解包 bnk 文件
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\135561656.bnk" -u -o "C:\unpacked_bnk_files"
```

### 3. 替换 `.pck` 内的文件（核心功能）

这是本工具为 SDDE 定制的核心功能。请严格按照以下步骤操作。

**第一步：准备替换文件**

1.  创建一个主文件夹，用于存放所有待替换的文件（例如 `D:\my_replacements`）。
2.  在该主文件夹内，根据你要替换的文件类型，创建 `bnk` 和/或 `wem` 子文件夹。
3.  将你要用于替换的 `.bnk` 或 `.wem` 文件放入对应的子文件夹中。
4.  **关键：** 将这些文件的文件名修改为你**在第一步中查到的 Index 号**。
    -   例如，要替换 `BnkIndex[1]`，就把你的新 bnk 文件命名为 `1.bnk`，并放入 `bnk` 文件夹。
    -   例如，要替换 `WemIndex[5]`，就把你的新 wem 文件命名为 `5.wem`，并放入 `wem` 文件夹。

**目录结构示例:** 
```
D:\my_replacements\
├───bnk\
│   └───1.bnk
└───wem\
    └───5.wem
```

**第二步：执行替换命令**

使用 `-r` (replace) 参数，并同时提供源文件、替换文件目录和新文件的输出路径。

**指令格式:** 
```bash
wwiseutil_SDDE.exe -f "<源pck文件路径>" -r -t "<你的替换文件主目录>" -o "<新生成的pck文件路径>"
```

**示例:** 
```bash
wwiseutil_SDDE.exe -f "C:\SDDE\Data\Audio\SD2\sfx.pck" -r -t "D:\my_replacements" -o "C:\SDDE\Data\Audio\SD2\sfx_new.pck"
```

命令执行成功后，`sfx_new.pck` 就是包含了你修改后内容的新文件。你可以将其重命名回`sfx.pck`并替换游戏原文件来进行测试。

## 致谢

- 感谢 **hpxro7** 创建了原版的 [wwiseutil](https://github.com/hpxro7/wwiseutil)。

## 许可证

本项目基于 GNU General Public License v3.0。详情请查看 `LICENSE` 文件。
