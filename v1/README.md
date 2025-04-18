# AI Coder

This script leverages Google's Gemini large language models to analyze and potentially modify source code files based on a user-provided prompt. It can read multiple files, send their content along with instructions to the Gemini API, calculate token usage and estimated cost, and either display the AI's response or attempt to modify the original files in-place.

**Features:**

*   Process multiple source files.
*   Integrates with Google Gemini models (configurable).
*   Uses Application Default Credentials (ADC) for authentication.
*   Calculates estimated API usage cost based on token count.
*   Optional in-place file modification (**Use with extreme caution!**).
*   Saves final prompt and API responses to temporary files for inspection.
*   Provides detailed logging with timestamps and color coding.
*   Attempts to open the final response in a web browser for easy viewing (when not modifying in-place or if modification fails).

## Requirements

*   Python 3.x
*   Google Cloud SDK installed and configured (`gcloud auth application-default login`).
*   Required Python packages:
    ```bash
    pip install google-generativeai google-auth colorama
    ```
*   Access granted for your ADC credentials to use the Gemini API on a Google Cloud project.

## Configuration

*   **Model Name:** Modify `MODEL_NAME` in `coder.py` to use a different Gemini model.
*   **Default Prompt:** Change `DEFAULT_USER_PROMPT` for a different default instruction if no `--prompt` is provided.
*   **Default Files:** Adjust `DEFAULT_SOURCE_DIR` and `DEFAULT_SOURCE_FILES` if you want the script to run without any file arguments.
*   **Pricing:** Update the `PRICE_INPUT_*` and `PRICE_OUTPUT_*` constants if Google's pricing changes or if you use a model with different rates.

## Usage

```bash
python coder.py [options] [file1 file2 ...]
```

**Arguments:**

*   `files` (optional): Positional arguments specifying the paths to the source files to process.
*   `--file-list <path>`: Path to a file containing a list of source file paths (one per line). Overrides any files provided positionally.
*   `--prompt "<prompt text>"`: The specific instruction or question to ask the Gemini API about the provided files. Defaults to the `DEFAULT_USER_PROMPT` in the script.
*   `--inplace`: **DANGEROUS!** If set, the script will attempt to parse the Gemini response (expecting a specific format) and overwrite the original source files. **BACK UP YOUR FILES FIRST!**

## Examples

1.  **Analyze two specific files with the default prompt:**
    ```bash
    python coder.py path/to/your/file1.go path/to/another/file2.py
    ```
    *This will read the content of `file1.go` and `file2.py`, send it to Gemini with the default prompt, print the AI's response to the console, show usage/cost info, and attempt to open the full response text (saved in `/tmp`) in your browser.*

2.  **Analyze files listed in a file with a custom prompt:**
    *   Create a file named `my_files.txt`:
        ```
        /project/src/main.java
        /project/src/utils.java
        # This is a comment, it will be ignored
        /project/tests/test_main.java
        ```
    *   Run the script:
        ```bash
        python coder.py --file-list my_files.txt --prompt "Refactor these Java files to improve exception handling."
        ```
    *This will process the three Java files listed in `my_files.txt`, using your custom prompt.*

3.  **Attempt in-place modification (Use with Caution!):**
    ```bash
    python coder.py --inplace --prompt "In the following Go files, remove all logging statements that use the 'fmt' package. Respond ONLY with the complete modified file content in the specified BEGIN/END format." service.go helpers.go api.go
    ```
    *   **WARNING:** This command attempts to directly overwrite `service.go`, `helpers.go`, and `api.go`.
    *   It sends the files' content to Gemini with a specific instruction.
    *   **Crucially**, for `--inplace` to work, your prompt MUST instruct the AI to respond *only* with the modified file content, wrapped in `--- BEGIN of /absolute/path/to/file ---` and `--- END of /absolute/path/to/file ---` markers for *each* file. The script parses this specific format.
    *   **BACK UP YOUR FILES BEFORE RUNNING THIS.** If the AI's response format is incorrect, or the modifications are wrong, your files could be corrupted or lost.
    *   The script will report which files were successfully modified and show any errors encountered during parsing or writing.

## Output

The script provides:

1.  **Console Logging:** Step-by-step progress, warnings, errors, and success messages.
2.  **API Response/Status:** Prints the AI's response text (if not `--inplace`) or a summary of the inplace modification attempt.
3.  **Usage Information:** Displays estimated input/output tokens, API call duration, and calculated cost.
4.  **Temporary Files:** Saves the exact prompt sent (`gemini_final_prompt_*.txt`), the raw API response (`gemini_raw_response_*.txt`), and the cleaned/final response (`gemini_final_response_*.txt`) to your system's temporary directory (usually `/tmp`). Paths are logged to the console.
5.  **Browser:** Attempts to open the final response file automatically for review (unless `--inplace` runs without errors).

## Troubleshooting

*   **Authentication Errors:** Ensure you have run `gcloud auth application-default login` and that the logged-in user/service account has permissions for the Gemini API (e.g., "Vertex AI User" role).
*   **File Not Found Errors:** Double-check the file paths provided directly or within the `--file-list` file. The script checks for existence before proceeding.
*   **In-place Modification Failure:**
    *   Check the temporary response file (`gemini_final_response_*.txt`) to see if the AI followed the required `--- BEGIN/END ---` format with *absolute paths*.
    *   Refine your `--prompt` to be extremely clear about the required output format.
    *   Ensure the script has write permissions for the target files.
*   **Cost:** Monitor your Google Cloud billing, as the script provides only an *estimate* based on token counts reported by the API.