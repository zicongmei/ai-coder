
# AI Coder

This application leverages Google's Gemini large language models to analyze and potentially modify source code files based on a user-provided prompt. It can read multiple files, send their content along with instructions to the Gemini API, calculate token usage, and either display the AI's response or attempt to modify the original files in-place. The application also supports generating an HTML version of the Gemini response for easier viewing.

**Features:**

*   Process multiple source files, preferably specified via a file list (`--file-list`).
*   Integrates with Google Gemini models (configurable via `--flash` flag, defaults to `gemini-2.5-pro`).
*   **Prioritized Authentication:** Uses Google Cloud Application Default Credentials (ADC) by default, or an API key provided via the `GEMINI_API_KEY` environment variable.
*   Calculates estimated API usage token count.
*   Optional in-place file modification (`--inplace`) using a specific text format requiring **absolute file paths** (**Use with extreme caution!**).
*   **Google Search Integration:** Optionally enable the Google Search tool (`--google-search`) to allow the model to fetch real-time information.
*   Saves the generated prompt (`ai_prompt_*.txt`) and the raw AI output (`ai_raw_output_*.txt`) to temporary files (in `/tmp`) for inspection.
*   Provides detailed logging using `glog`, outputting to stderr by default (and optionally to files).
*   Converts AI's raw response (which is often Markdown) to HTML for non-inplace operations.
*   Attempts to open the final HTML response (or the raw text output if HTML conversion fails/skipped) in a web browser for easy viewing (when not modifying in-place).

## Requirements

*   **Go 1.20+**
*   **Google Cloud SDK installed and configured (`gcloud auth application-default login`) for the primary authentication method.**
*   Access granted for your ADC credentials or API key to use the Gemini API on a Google Cloud project.

## Building and Running

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/zicongmei/ai-coder.git
    cd ai-coder/v2
    ```

2.  **Build the executable:**
    ```bash
    go build -o coder .
    ```
    This will create an executable named `coder` in the current directory.

3.  **Run the application:**
    You can run it directly using the built executable:
    ```bash
    ./coder --prompt "<your_prompt>" --file-list <path_to_file_list> [other_options]
    ```
    Or, you can run it using `go run` (which will compile and run in one step):
    ```bash
    go run . --prompt "<your_prompt>" --file-list <path_to_file_list> [other_options]
    ```

## Configuration

*   **Model Name:** The model used is `gemini-2.5-pro` by default. To use the faster `gemini-2.5-flash` model, provide the `--flash` flag.
*   **Authentication:**
    *   By default, the application attempts to use Google Cloud Application Default Credentials (ADC). Ensure you have run `gcloud auth application-default login`.
    *   Alternatively, set the `GEMINI_API_KEY` environment variable with your Gemini API key:
        ```bash
        export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
        ```
*   **Logging:** The application uses `glog`. By default, logs go to stderr (`-alsologtostderr=true`). You can control verbosity with `-v` (e.g., `-v=2`). See `glog` documentation for more advanced logging options.

## Usage

The recommended way to use the application is with a file list for specifying source files and relying on Google Cloud ADC or the `GEMINI_API_KEY` environment variable for authentication. In-place modification should be used cautiously.

```bash
./coder --prompt "<prompt_text>" --file-list <path_to_file_list> [--inplace] [--flash] [--google-search]
# Or using go run:
go run . --prompt "<prompt_text>" --file-list <path_to_file_list> [--inplace] [--flash] [--google-search]
```

**Key Arguments:**

*   `--prompt "<prompt text>"` (**REQUIRED**): The base prompt/instruction for the Gemini API. Format instructions for in-place modification are added automatically by the application.
*   `--file-list <path>` (**REQUIRED**): Path to a file containing a list of source file paths (one per line).
*   `--inplace` (optional, **DANGEROUS!**): If set, the application will attempt to parse the Gemini response (expecting a specific format with **absolute file paths**) and overwrite the original source files. **BACK UP YOUR FILES FIRST!**
*   `--flash` (optional): If set, uses the `gemini-2.5-flash` model for potentially faster, cheaper responses, at the possible expense of quality. By default, `gemini-2.5-pro` is used.
*   `--google-search` (optional): If set, enables the Google Search tool for the Gemini model, allowing it to retrieve external information (e.g., current documentation, recent events) to ground its response.

## Examples

1.  **Analyze files listed in `my_sources.txt` (using gcloud ADC):**
    *   Create `my_sources.txt`:
        ```
        /path/to/your/project/file1.go
        /path/to/your/project/module/file2.py
        another/absolute/path/to/file3.java
        ```
    *   Run the application:
        ```bash
        ./coder --prompt "Explain the purpose of these files and identify potential bugs." --file-list my_sources.txt
        ```
    *This will read the content of the files listed in `my_sources.txt`, send it to Gemini with your prompt (using your gcloud ADC for authentication), print the AI's response, show token usage info, convert the response to HTML, and attempt to open the HTML file (saved in `/tmp`) in your browser.*

2.  **Attempt in-place modification for files in `refactor_list.txt` (using gcloud ADC and flash model):**
    *   Create `refactor_list.txt`:
        ```
        /app/src/service.go
        /app/src/helpers.go
        ```
    *   Run the application:
        ```bash
        ./coder --inplace --flash --prompt "In these Go files, refactor all functions that return an error to also return a boolean indicating success." --file-list refactor_list.txt
        ```
    *   **WARNING:** This command attempts to directly overwrite the files listed in `refactor_list.txt`.
    *   It sends the files' content to Gemini with your specific instruction, relying on gcloud ADC and using the `gemini-2.5-flash` model.
    *   The application automatically adds detailed format instructions to your prompt when `--inplace` is used, emphasizing the use of absolute paths for the AI's response.
    *   **BACK UP YOUR FILES BEFORE RUNNING THIS.**
    *   The application will report which files were successfully modified. If errors occur, it might open the raw response file in a browser.

3.  **Analyze files using an API Key (via environment variable):**
    ```bash
    export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
    ./coder --prompt "Review this Python script for security vulnerabilities." --file-list /scripts/script_list.txt
    ```
    *This demonstrates providing an API key via an environment variable for authentication instead of gcloud ADC.*

## Output

The application provides:

1.  **Console Logging:** Step-by-step progress, warnings, errors, and success messages, output to stderr by default.
2.  **API Response/Status:** Prints a summary of the in-place modification attempt. If not in in-place mode, the AI's response will be formatted as HTML and opened in a browser.
3.  **Usage Information:** Displays estimated input/output tokens and API call duration.
4.  **Temporary Files:** Saves the exact prompt sent (`ai_prompt_*.txt`) and the raw AI output (`ai_raw_output_*.txt`) to your system's temporary directory (usually `/tmp`). For non-inplace operations, an HTML version of the response (`ai_raw_response_*.html`) is also generated there. Paths are logged to the console.
5.  **Browser:**
    *   For non-inplace operations, attempts to open the generated HTML file automatically.
    *   For inplace operations, if modification is successful without errors, it typically skips opening any file. If there are errors during the inplace process, it may attempt to open the raw response file.

## Troubleshooting

*   **Authentication Errors:**
    *   **gcloud ADC (Default):** Ensure you have run `gcloud auth application-default login` and that the logged-in user/service account has permissions for the Gemini API (e.g., "Vertex AI User" role or similar).
    *   **`GEMINI_API_KEY`:** Ensure your environment variable `GEMINI_API_KEY` is set correctly and the key is valid with necessary permissions.
*   **Command Not Found (`./coder`):** Ensure you have built the executable using `go build -o coder .` and that you are in the directory where the `coder` executable was created.
*   **File Not Found Errors:** Double-check the file paths provided in the `--file-list` file. The application checks for existence and uses absolute paths internally.
*   **In-place Modification Failure:**
    *   Check the temporary raw AI output file (`ai_raw_output_*.txt` in `/tmp`) to see if the AI followed the required `--- Start of File: /absolute/path/to/file ---\n...--- End of File: /absolute/path/to/file ---\n` format with correct **absolute paths**.
    *   Refine your `--prompt` to be extremely clear about the required output format. The application already adds specific instructions for the AI when `--inplace` is used.
    *   Ensure the application has write permissions for the target files.