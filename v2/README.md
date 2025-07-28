# AI Coder

This script leverages Google's Gemini large language models to analyze and potentially modify source code files based on a user-provided prompt. It can read multiple files, send their content along with instructions to the Gemini API, calculate token usage and estimated cost (with tiered pricing), and either display the AI's response or attempt to modify the original files in-place. The script also supports generating an HTML version of the Gemini response for easier viewing.

**Features:**

*   Process multiple source files, preferably specified via a file list (`--file-list`).
*   Integrates with Google Gemini models (configurable, defaults to `gemini-2.5-pro-preview-05-06`).
*   **Prioritized Authentication:** Uses Google Cloud Application Default Credentials (ADC) by default.
*   Calculates estimated API usage cost based on token count using a tiered pricing model.
*   Optional in-place file modification (`--inplace`) using a specific text format requiring **absolute file paths** (**Use with extreme caution!**).
*   Saves final prompt, raw API response, cleaned response, and generated HTML report to temporary files (in `/tmp`) for inspection.
*   Provides detailed, color-coded logging with timestamps for each step.
*   Converts Gemini's markdown response to HTML for non-inplace operations.
*   Attempts to open the final HTML response (or text response if HTML conversion fails/skipped) in a web browser for easy viewing (when not modifying in-place successfully, or if modification has errors).
*   Supports providing a direct API key (`--api-key`) as an alternative authentication method.

## Requirements

*   Python 3.x
*   **Google Cloud SDK installed and configured (`gcloud auth application-default login`) for the primary authentication method.**
*   Required Python packages:
    ```bash
    pip install google-generativeai google-auth colorama markdown
    ```
*   Access granted for your ADC credentials or API key to use the Gemini API on a Google Cloud project.

## Configuration

*   **Model Name:** Modify `MODEL_NAME` in `coder.py` to use a different Gemini model (default: `gemini-2.5-pro-preview-05-06`).
*   **Default Files:** Adjust `DEFAULT_SOURCE_DIR` and `DEFAULT_SOURCE_FILES` in `coder.py` if you want the script to run with default files when no file arguments are provided (though using `--file-list` or direct file arguments is recommended).
*   **Pricing:** Update the `TOKEN_LIMIT_TIER_1`, `PRICE_INPUT_TIER_1`, `PRICE_INPUT_TIER_2`, `PRICE_OUTPUT_TIER_1`, and `PRICE_OUTPUT_TIER_2` constants in `coder.py` if Google's pricing changes or if you use a model with different rates. These are used for estimated cost calculation.

## Usage

The recommended way to use the script is with a file list for specifying source files and relying on Google Cloud ADC for authentication. In-place modification should be used cautiously.

```bash
python coder.py --prompt "<prompt_text>" --file-list <path_to_file_list> [--inplace] [other_options]
```

**Key Arguments:**

*   `--prompt "<prompt text>"` (**REQUIRED**): The base prompt/instruction for the Gemini API. Format instructions for in-place modification are added automatically by the script.
*   `--file-list <path>` (Recommended): Path to a file containing a list of source file paths (one per line). This overrides any files provided positionally.
*   `--inplace` (optional, **DANGEROUS!**): If set, the script will attempt to parse the Gemini response (expecting a specific format with **absolute file paths**) and overwrite the original source files. **BACK UP YOUR FILES FIRST!**
*   `files` (optional): Positional arguments specifying the paths to the source files to process. Used if `--file-list` is not provided.
*   `--api-key <key>` (optional, alternative authentication): Your Gemini API key. Use this if you are not using Google Cloud ADC.

## Examples

1.  **Analyze files listed in `my_sources.txt` (using gcloud ADC):**
    *   Create `my_sources.txt`:
        ```
        /path/to/your/project/file1.go
        /path/to/your/project/module/file2.py
        another/relative/path/to/file3.java
        ```
    *   Run the script:
        ```bash
        python coder.py --prompt "Explain the purpose of these files and identify potential bugs." --file-list my_sources.txt
        ```
    *This will read the content of the files listed in `my_sources.txt`, send it to Gemini with your prompt (using your gcloud ADC for authentication), print the AI's response, show usage/cost info, convert the response to HTML, and attempt to open the HTML file (saved in `/tmp`) in your browser.*

2.  **Attempt in-place modification for files in `refactor_list.txt` (using gcloud ADC):**
    *   Create `refactor_list.txt`:
        ```
        /app/src/service.go
        /app/src/helpers.go
        ```
    *   Run the script:
        ```bash
        python coder.py --inplace --prompt "In these Go files, refactor all functions that return an error to also return a boolean indicating success. Respond ONLY with the complete modified file content for each file, using the specified BEGIN/END format with their absolute paths." --file-list refactor_list.txt
        ```
    *   **WARNING:** This command attempts to directly overwrite the files listed in `refactor_list.txt`.
    *   It sends the files' content to Gemini with your specific instruction, relying on gcloud ADC.
    *   **Crucially**, for `--inplace` to work, your prompt MUST instruct the AI to respond *only* with the modified file content, wrapped in `--- BEGIN of /absolute/path/to/file ---` and `--- END of /absolute/path/to/file ---` markers for *each* file. The script automatically adds detailed format instructions to your prompt when `--inplace` is used, emphasizing the use of absolute paths.
    *   **BACK UP YOUR FILES BEFORE RUNNING THIS.**
    *   The script will report which files were successfully modified. If errors occur, it might open the raw response file.

3.  **Analyze specific files using an API Key (alternative authentication):**
    ```bash
    python coder.py --prompt "Review this Python script for security vulnerabilities." --api-key "YOUR_GEMINI_API_KEY" /scripts/myscript.py
    ```
    *This demonstrates providing files directly and using the `--api-key` for authentication instead of gcloud ADC.*

## Output

The script provides:

1.  **Console Logging:** Step-by-step progress, warnings, errors, and success messages, all color-coded and timestamped.
2.  **API Response/Status:** Prints the AI's response text (if not `--inplace`) or a summary of the inplace modification attempt.
3.  **Usage Information:** Displays estimated input/output tokens, API call duration, and calculated cost based on tiered pricing.
4.  **Temporary Files:** Saves the exact prompt sent (`gemini_final_prompt_*.txt`), the raw API response (`gemini_raw_response_*.txt`), the cleaned/final response (`gemini_final_response_*.txt`), and the HTML version of the response (`gemini_html_result_*.html`) to your system's temporary directory (usually `/tmp`). Paths are logged to the console.
5.  **Browser:**
    *   For non-inplace operations, attempts to open the generated HTML file (or the text response if HTML fails) automatically.
    *   For inplace operations, if modification is successful without errors, it typically skips opening any file. If there are errors during the inplace process, it may attempt to open the raw response file.

## Troubleshooting

*   **Authentication Errors:**
    *   **gcloud ADC (Default):** Ensure you have run `gcloud auth application-default login` and that the logged-in user/service account has permissions for the Gemini API (e.g., "Vertex AI User" role or similar).
    *   **`--api-key`:** Ensure your API key is valid and has the necessary permissions.
*   **`markdown` Library Missing:** The script requires the `markdown` library for HTML conversion. If not installed, the script will print an error and exit. Install it with `pip install markdown`.
*   **File Not Found Errors:** Double-check the file paths provided in the `--file-list` file or as direct arguments. The script checks for existence and uses absolute paths internally.
*   **In-place Modification Failure:**
    *   Check the temporary response file (`gemini_final_response_*.txt` in `/tmp`) to see if the AI followed the required `--- BEGIN of /absolute/path/to/file --- ... --- END of /absolute/path/to/file ---` format with correct **absolute paths**.
    *   Refine your `--prompt` to be extremely clear about the required output format. The script already adds specific instructions for the AI when `--inplace` is used.
    *   Ensure the script has write permissions for the target files.
*   **Cost:** Monitor your Google Cloud billing, as the script provides only an *estimate* based on token counts reported by the API and predefined pricing tiers in the script.