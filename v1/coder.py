import google.generativeai as genai
import google.auth
import os
import argparse # Added for command-line arguments
import re # Now used for parsing the text format
import json # Kept for potential future use or other parts, but not for inplace response
import tempfile # Added for temporary file creation
import datetime # Added for potential timestamping if needed (using tempfile now)
from typing import Tuple, Dict, Any # Added for type hinting the return value
import colorama # Added for colored output
from colorama import Fore, Style, init # Import specific colorama components
import time # Import time for basic timestamping and duration calculation
import webbrowser # Added to open files in browser

# --- Configuration ---
# Specify the Gemini model you want to use
# MODEL_NAME = "gemini-2.0-flash"
MODEL_NAME = "gemini-2.5-pro-preview-03-25"
# Define the prompt or question you want to ask about the file contents
# This prompt will be *appended* with format instructions when --inplace is used.
DEFAULT_USER_PROMPT = "Return new files for removing the gateway, gcp backend, service entry, virtual service from the reconciling the service."

# Placeholder paths - **Replace these with your actual paths**
# Used only if NO files are provided via command line OR --file-list
DEFAULT_SOURCE_DIR = "/usr/local/google/home/zicong/code/src/user/zicong/cloudrun-controller/internal/controller/"
DEFAULT_SOURCE_FILES = [
    DEFAULT_SOURCE_DIR + "service_controller.go",
    DEFAULT_SOURCE_DIR + "service.go",
    DEFAULT_SOURCE_DIR + "utils.go",
]

# --- Pricing Configuration ---
# Based on user-provided info for gemini-2.5-pro-preview-03-25 (assumed)
TOKEN_LIMIT_TIER_1 = 200000 # Tokens
# Price per 1 Million Tokens
PRICE_INPUT_TIER_1 = 1.25 # Prompts <= 200k tokens
PRICE_INPUT_TIER_2 = 2.50 # Prompts > 200k tokens
PRICE_OUTPUT_TIER_1 = 10.00 # Prompts <= 200k tokens
PRICE_OUTPUT_TIER_2 = 15.00 # Prompts > 200k tokens

# --- Logging Helper ---
def log_info(message):
    """Prints an informational log message."""
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.CYAN}[{timestamp} INFO]{Style.RESET_ALL} {message}")

def log_step(message):
    """Prints a step marker log message."""
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.BLUE}[{timestamp} STEP]{Style.RESET_ALL} {message}")

def log_warn(message):
    """Prints a warning log message."""
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.YELLOW}[{timestamp} WARN]{Style.RESET_ALL} {message}")

def log_error(message):
    """Prints an error log message."""
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.RED}[{timestamp} ERROR]{Style.RESET_ALL} {message}")

def log_success(message):
    """Prints a success log message."""
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.GREEN}[{timestamp} OK]{Style.RESET_ALL} {message}")


# --- Authentication ---
log_step("Attempting Authentication...")
try:
    credentials, project_id = google.auth.default()
    genai.configure(credentials=credentials)
    log_success(f"Successfully authenticated using ADC for project: {project_id}")
except google.auth.exceptions.DefaultCredentialsError as e:
    log_error(f"Authentication failed. Please run 'gcloud auth application-default login'. Error: {e}")
    exit(1) # Exit if authentication fails
except Exception as e:
    log_error(f"An unexpected error occurred during authentication: {e}")
    exit(1)

# --- File Processing and Modification Function ---
def process_files_with_gemini(file_paths: list[str], prompt: str, inplace: bool) -> Tuple[str, Dict[str, Any]]:
    """
    Reads content from EXISTING files, sends it to Gemini (requesting specific text format for inplace),
    saves the raw/cleaned responses, calculates token usage/cost/duration, and either returns the
    response or attempts to modify files inplace based on the text format using ABSOLUTE paths.

    Args:
        file_paths: A list of paths to EXISTING source files.
        prompt: The user's question or instruction for the API.
        inplace: If True, request text format and attempt to modify files inplace.

    Returns:
        A tuple containing:
        - The text response or status message from the Gemini API (str).
        - A dictionary with usage info: {
            'input_tokens': int,
            'output_tokens': int,
            'cost': float,
            'api_call_duration': float | None,   # Duration of the generate_content call
            'prompt_file_path': str | None,      # Path to saved final prompt
            'raw_response_file': str | None,      # Path to saved raw response
            'cleaned_response_file': str | None, # Path to saved cleaned response (may be same as raw if no cleaning needed)
            'error': str | None                  # Error message if any step failed
          }
    """
    log_step(f"Entering process_files_with_gemini for {len(file_paths)} files.")
    files_content_map = {} # Store original content mapped to absolute path
    # base_to_abs_path_map removed - no longer needed
    combined_content_for_prompt = ""
    # base_filenames removed - no longer needed
    usage_info = { # Initialize usage info
        'input_tokens': 0,
        'output_tokens': 0,
        'cost': 0.0,
        'api_call_duration': None, # Initialize duration
        'prompt_file_path': None,
        'raw_response_file': None,
        'cleaned_response_file': None,
        'error': None
    }
    log_step("Starting file reading loop...")
    for file_path in file_paths:
        try:
            abs_file_path = os.path.abspath(file_path) # Use absolute path as the key
            # base_name = os.path.basename(abs_file_path) # Base name no longer needed for mapping
            if not os.path.exists(abs_file_path):
                 log_warn(f"File path provided but not found - {abs_file_path}. Skipping.")
                 continue
            log_info(f"Reading file: {abs_file_path}")
            with open(abs_file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                # Store content using absolute path as the key
                if abs_file_path in files_content_map:
                    # This case should ideally not happen if input list has unique absolute paths
                    log_warn(f"Duplicate absolute file path '{abs_file_path}' detected in input. Using content from the last occurrence.")
                files_content_map[abs_file_path] = content
                # Construct prompt markers using absolute path
                marker_start = f"\n--- Start of File: {abs_file_path} ---\n"
                marker_end = f"\n--- End of File: {abs_file_path} ---\n"
                combined_content_for_prompt += marker_start + content + marker_end
        except Exception as e:
            log_error(f"Error reading file {file_path}: {e}. Skipping.")
    log_step("Finished file reading loop.")


    if not files_content_map:
        usage_info['error'] = "No content could be read."
        err_msg = "Error: No content could be read from the specified files (check paths and permissions)."
        log_error(err_msg)
        return err_msg, usage_info

    # --- Construct the final prompt ---
    log_step("Constructing final prompt...")
    final_prompt = f"{prompt}\n\n{combined_content_for_prompt}"

    if inplace:
        # *** MODIFIED INSTRUCTION FOR TEXT FORMAT using ABSOLUTE PATHS ***
        text_format_instruction = (
            "\n\nIMPORTANT: Respond ONLY with the complete, modified content for each file, "
            "formatted exactly as follows, using the ABSOLUTE file paths provided:\n"
        )
        # Iterate through the absolute paths (keys of the map)
        absolute_paths_in_prompt = list(files_content_map.keys())
        for abs_path in absolute_paths_in_prompt:
             text_format_instruction += (
                 f"--- BEGIN of {abs_path} ---\n" # Use absolute path
                 f"{{content for {abs_path}}}\n" # Placeholder notation
                 f"--- END of {abs_path} ---\n"   # Use absolute path
             )
        text_format_instruction += (
            "\nDo not include any introductory text, explanations, or other formatting outside "
            "of these BEGIN/END blocks. Ensure the ABSOLUTE file paths in the BEGIN/END markers "
            f"match the requested files: {', '.join(absolute_paths_in_prompt)}."
        )
        final_prompt += text_format_instruction
        log_info("Added TEXT format instruction to prompt for inplace modification (using absolute paths).")
    log_step("Final prompt constructed.")
    log_info(f"Constructed final prompt size: {len(final_prompt)} characters.")

    # --- Save Final Prompt Before Sending ---
    log_step("Attempting to save final prompt to /tmp...")
    try:
        with tempfile.NamedTemporaryFile(
            mode='w', encoding='utf-8',
            prefix='gemini_final_prompt_', suffix='.txt',
            dir='/tmp', delete=False # Keep the file after closing
        ) as tmp_prompt_file:
            tmp_prompt_file.write(final_prompt)
            prompt_file_path = tmp_prompt_file.name
            usage_info['prompt_file_path'] = prompt_file_path # Store path
            log_info(f"Saved final prompt to: {prompt_file_path}")
    except Exception as save_prompt_e:
        log_warn(f"Failed to save final prompt to /tmp. Error: {save_prompt_e}")
    log_step("Finished saving final prompt.")
    # --- End Save Final Prompt ---


    # --- Gemini API Interaction ---
    raw_response_text = ""
    response = None # Initialize response object
    api_duration = None # Initialize duration
    try:
        # Initialize model here to use count_tokens before the main call
        model = genai.GenerativeModel(MODEL_NAME)

        # --- Count Input Tokens (Client-side Estimation) ---
        log_step("Estimating input tokens (client-side)...")
        try:
            count_response = model.count_tokens(final_prompt)
            input_token_estimate = count_response.total_tokens
            log_info(f"Estimated input tokens (client-side count): {input_token_estimate}")
        except Exception as count_e:
            log_warn(f"Could not estimate input tokens client-side: {count_e}")
        # --- End Count Input Tokens ---

        log_step(f"Sending prompt to Gemini model: {MODEL_NAME}...")
        start_time = time.monotonic() # Record start time
        # Send the request to the API
        response = model.generate_content(final_prompt)
        end_time = time.monotonic() # Record end time
        api_duration = end_time - start_time # Calculate duration
        usage_info['api_call_duration'] = api_duration # Store duration

        raw_response_text = response.text # Get raw text immediately
        log_success(f"Received response from Gemini. Duration: {api_duration:.3f} seconds") # Log duration


        # --- Process Usage Metadata ---
        log_step("Processing usage metadata...")
        input_tokens = 0
        output_tokens = 0
        if response and hasattr(response, 'usage_metadata'):
            input_tokens = response.usage_metadata.prompt_token_count
            output_tokens = response.usage_metadata.candidates_token_count
            usage_info['input_tokens'] = input_tokens
            usage_info['output_tokens'] = output_tokens
            log_info(f"API Usage: Input Tokens = {input_tokens}, Output Tokens = {output_tokens}")

            # --- Calculate Cost ---
            log_step("Calculating estimated cost...")
            if input_tokens > TOKEN_LIMIT_TIER_1:
                input_price = PRICE_INPUT_TIER_2
                output_price = PRICE_OUTPUT_TIER_2
                log_info(f"Input tokens ({input_tokens}) > {TOKEN_LIMIT_TIER_1}, using Tier 2 pricing.")
            else:
                input_price = PRICE_INPUT_TIER_1
                output_price = PRICE_OUTPUT_TIER_1
                log_info(f"Input tokens ({input_tokens}) <= {TOKEN_LIMIT_TIER_1}, using Tier 1 pricing.")

            input_cost = (input_tokens / 1_000_000) * input_price
            output_cost = (output_tokens / 1_000_000) * output_price
            total_cost = input_cost + output_cost
            usage_info['cost'] = total_cost
            log_info(f"Calculated Cost: Input=${input_cost:.6f}, Output=${output_cost:.6f}, Total=${total_cost:.6f}")
            log_step("Finished cost calculation.")
            # --- End Calculate Cost ---

        else:
            log_warn("Usage metadata not found in response. Cannot calculate exact tokens or cost.")
            usage_info['error'] = "Usage metadata missing."
        log_step("Finished processing usage metadata.")


        # --- Save Raw Response ---
        log_step("Attempting to save raw response...")
        try:
            with tempfile.NamedTemporaryFile(
                mode='w', encoding='utf-8',
                prefix='gemini_raw_response_', suffix='.txt',
                dir='/tmp', delete=False
            ) as tmp_file:
                tmp_file.write(raw_response_text)
                raw_response_path = tmp_file.name
                usage_info['raw_response_file'] = raw_response_path # Store path
                log_info(f"Saved raw Gemini response to: {raw_response_path}")
        except Exception as save_e:
            log_warn(f"Failed to save raw Gemini response to /tmp. Error: {save_e}")
        log_step("Finished saving raw response.")
        # --- End Save Raw Response ---


        # Clean potential markdown/formatting *if not inplace*.
        log_step("Cleaning response text (if not inplace)...")
        response_text = raw_response_text # Start with raw
        if not inplace:
            response_text = raw_response_text.strip()
            cleaned = False
            # Basic markdown code block removal (might need adjustment)
            if response_text.startswith("```") and response_text.endswith("```"):
                 # Find first newline after initial ```
                 first_newline = response_text.find('\n')
                 if first_newline != -1:
                     response_text = response_text[first_newline+1:-3].strip()
                     cleaned = True
                 else: # Handle case like ```content```
                     response_text = response_text[3:-3].strip()
                     cleaned = True

            if cleaned:
                log_info("Removed potential markdown code fences from response (for non-inplace).")
        else:
            log_info("Skipping general cleaning for inplace mode (expecting specific absolute path format).")
        log_step("Finished cleaning response text.")

        # --- Save Cleaned Response ---
        log_step("Attempting to save final response text...")
        try:
            # Create another temporary file for the final version
            with tempfile.NamedTemporaryFile(
                mode='w', encoding='utf-8',
                prefix='gemini_final_response_', suffix='.txt', # Use 'final' prefix
                dir='/tmp', delete=False
            ) as tmp_file:
                tmp_file.write(response_text)
                cleaned_response_path = tmp_file.name
                usage_info['cleaned_response_file'] = cleaned_response_path # Store path
                log_info(f"Saved final Gemini response text to: {cleaned_response_path}")
        except Exception as save_e:
            log_warn(f"Failed to save final Gemini response text to /tmp. Error: {save_e}")
        log_step("Finished saving final response text.")
        # --- End Save Cleaned Response ---

    except Exception as e:
        error_msg = f"An error occurred while calling the Gemini API: {e}"
        log_error(error_msg)
        usage_info['error'] = str(e)
        # Also store duration if available even on error
        if api_duration is not None:
             usage_info['api_call_duration'] = api_duration
        return f"Error: API call failed - {e}", usage_info # Return error and usage info

    # --- Inplace Modification Logic (using Absolute Path Text Parsing) ---
    if inplace:
        log_step("Starting inplace modification process (parsing absolute path text format)...")
        modified_files_count = 0
        errors = []
        # Map to store extracted content {absolute_path: content}
        extracted_content_map: Dict[str, str] = {}

        if not response_text:
             err_msg = "Fatal Error: Received empty response from Gemini. Cannot parse text format."
             log_error(err_msg)
             errors.append(err_msg)
        else:
            log_step("Attempting to parse text response using BEGIN/END markers (expecting absolute paths)...")
            # Regex to find blocks: --- BEGIN of (absolute_path) --- \n {content} \n --- END of absolute_path ---
            # Pattern remains the same, but interpretation of group(1) changes
            pattern = re.compile(r"--- BEGIN of (.*?) ---\s*(.*?)\s*--- END of \1 ---", re.DOTALL)
            matches = pattern.finditer(response_text)

            found_matches = False
            for match in matches:
                found_matches = True
                # Group 1 is now expected to be the absolute path
                abs_path_from_response = match.group(1).strip()
                new_content = match.group(2).strip() # Strip whitespace around content

                # Validate if the path from response was actually in the input
                if abs_path_from_response not in files_content_map:
                     log_warn(f"Response included block for path '{abs_path_from_response}', which was NOT in the original input files. Ignoring this block.")
                     errors.append(f"Extraneous content block for '{abs_path_from_response}' in response.")
                     continue # Skip this match

                if abs_path_from_response in extracted_content_map:
                    log_warn(f"Duplicate BEGIN/END block found for absolute path '{abs_path_from_response}' in response. Using the last occurrence.")
                extracted_content_map[abs_path_from_response] = new_content
                log_info(f"Extracted content for absolute path: '{abs_path_from_response}'")

            if not found_matches:
                 err_msg = "Fatal Error: Could not find any valid '--- BEGIN of ... --- ... --- END of ... ---' blocks in the response."
                 log_error(err_msg)
                 errors.append(err_msg)
            elif not extracted_content_map and found_matches: # Found blocks, but all were invalid paths
                 err_msg = "Fatal Error: Found BEGIN/END blocks, but none matched the absolute paths of the input files."
                 log_error(err_msg)
                 errors.append(err_msg)
            elif extracted_content_map:
                 log_success(f"Successfully parsed text response, found valid content for {len(extracted_content_map)} file(s).")

            log_step("Finished text parsing attempt.")


        if not errors and extracted_content_map: # Proceed only if parsing didn't have fatal errors AND we have valid content
            log_step("Starting file overwrite loop (using validated absolute paths)...")
            # Iterate through the *original* files to ensure we only write to expected paths
            # and to check for missing blocks in the response.
            original_paths_processed = set()
            for abs_path in files_content_map.keys():
                if abs_path in extracted_content_map:
                    new_content = extracted_content_map[abs_path]
                    log_info(f"Validated extracted content for: {abs_path}")
                    try:
                        log_info(f"Overwriting file: {abs_path}")
                        with open(abs_path, 'w', encoding='utf-8') as f:
                            f.write(new_content)
                        modified_files_count += 1
                        original_paths_processed.add(abs_path)
                    except Exception as e:
                        error_msg = f"Error writing file {abs_path}: {e}"
                        log_error(error_msg)
                        errors.append(error_msg)
                else:
                    # This original file path was not found in the response blocks
                    log_warn(f"No valid '--- BEGIN/END ---' block found in response for absolute path '{abs_path}'. File not changed.")
                    errors.append(f"Missing content block for '{abs_path}' in response.")

            # Double check if any extracted paths were somehow missed (shouldn't happen with current logic, but safe)
            # for resp_path in extracted_content_map:
            #     if resp_path not in original_paths_processed:
            #         log_warn(f"Internal inconsistency: Path '{resp_path}' was extracted but not processed during overwrite. Ignoring.")
            #         errors.append(f"Internal inconsistency processing path '{resp_path}'.")

            log_step("Finished file overwrite loop.")

        # --- Report Results ---
        log_step("Compiling inplace modification results...")
        if not errors and modified_files_count == len(files_content_map):
             # Success only if no errors AND all original files were modified
             result_message = f"Inplace modification complete. {modified_files_count} file(s) modified based on text response (using absolute paths)."
             log_success(result_message)
        elif not errors and modified_files_count < len(files_content_map):
             # Partial success (e.g., some blocks missing from response)
             result_message = f"Inplace modification attempted. {modified_files_count} out of {len(files_content_map)} file(s) modified."
             result_message += "\nErrors/Warnings (check logs for details):\n"
             # Add specific warning about missing blocks if applicable
             missing_count = len(files_content_map) - modified_files_count
             result_message += f"- {missing_count} file(s) were not modified (missing content blocks in response).\n"
             result_message += "\n".join([f"- {e}" for e in errors if "Missing content block" not in e]) # Show other errors
             log_warn("Inplace modification finished partially (some files not modified).")
        else:
             # If there were errors (parsing, writing, etc.)
             result_message = f"Inplace modification attempted. {modified_files_count} file(s) modified."
             result_message += "\nErrors encountered during processing/writing:\n" + "\n".join([f"- {e}" for e in errors]) # Format errors
             if not extracted_content_map and any("Fatal Error" in err for err in errors):
                 result_message += "\n\n" + Fore.RED + "Text parsing failed or yielded no valid content. No files were modified."
             log_warn("Inplace modification finished with errors.") # Log warning if errors occurred

        return result_message, usage_info # Return status message and usage info

    else:
        # If not inplace, return the potentially cleaned response text and usage info
        log_step("Exiting process_files_with_gemini (inplace=False).")
        return response_text, usage_info # Return final text

# --- Main Execution Block ---
if __name__ == "__main__":
    # Initialize colorama - autoreset ensures styles apply only to the print statement
    init(autoreset=True)
    log_step("Script execution started.")

    parser = argparse.ArgumentParser(description="Process EXISTING files with the Gemini API, calculate cost, and optionally modify them inplace using a specific text format based on ABSOLUTE paths.")

    # --- Argument Parsing ---
    log_step("Parsing command line arguments...")
    parser.add_argument(
        "files",
        nargs='*', # 0 or more positional arguments for direct file paths
        default=None,
        help="Paths to the source files to process directly. Use --file-list instead to read paths from a file."
    )
    parser.add_argument(
        "--file-list",
        type=str,
        default=None, # Default to None, meaning it's not provided
        help="Path to a file containing a list of source file paths (one per line). Overrides any files provided directly."
    )
    parser.add_argument(
        "--inplace",
        action="store_true", # Sets to True if flag is present
        help="Modify the source files inplace based on Gemini text response format (using absolute paths). **USE WITH CAUTION**"
    )
    parser.add_argument(
        "--prompt",
        type=str,
        default=DEFAULT_USER_PROMPT,
        help="The base prompt/instruction for the Gemini API. Format instructions are added automatically for --inplace."
    )
    args = parser.parse_args()
    log_step("Finished parsing arguments.")
    log_info(f"Arguments: files={args.files}, file_list={args.file_list}, inplace={args.inplace}, prompt='{args.prompt[:50]}...'")


    # --- Determine Files to Process ---
    log_step("Determining source files to process...")
    source_files_to_check = []
    if args.file_list:
        log_info(f"Reading file list from: {args.file_list}")
        try:
            with open(args.file_list, 'r', encoding='utf-8') as f_list:
                for line in f_list:
                    stripped_line = line.strip()
                    # Ignore empty lines and comments
                    if stripped_line and not stripped_line.startswith('#'):
                        source_files_to_check.append(stripped_line)
            if not source_files_to_check:
                 log_warn(f"File list '{args.file_list}' was empty or only contained comments.")
                 log_error("Exiting because the specified file list is effectively empty.")
                 exit(1)
            log_info(f"Read {len(source_files_to_check)} file paths from list file.")

        except FileNotFoundError:
            log_error(f"File list not found at '{args.file_list}'. Exiting.")
            exit(1)
        except IOError as e:
            log_error(f"Error reading file list '{args.file_list}': {e}. Exiting.")
            exit(1)

    elif args.files: # User provided files directly
        source_files_to_check = args.files
        log_info(f"Using {len(source_files_to_check)} files provided directly via command line.")
    else: # No files provided via args or --file-list, use defaults
        source_files_to_check = DEFAULT_SOURCE_FILES
        log_info(f"No files provided, using default paths: {', '.join(source_files_to_check)}")
    log_step("Finished determining source files.")


    user_prompt = args.prompt

    if args.inplace:
         log_warn("\n--- Running in INPLACE mode ---")
         log_warn("WARNING: This will attempt to overwrite files based on Gemini's text output (using absolute paths).")
         log_warn("Ensure your prompt clearly asks for the desired modifications in the specified format.")
         log_warn("BACK UP YOUR FILES before proceeding if you haven't already.")
         log_warn("-------------------------------\n")

    # --- File Existence Check ---
    valid_files = []
    invalid_files = []
    log_step("Checking file existence...")
    for f in source_files_to_check:
        # Resolve to absolute path *before* checking existence
        abs_f = os.path.abspath(f)
        if os.path.exists(abs_f) and os.path.isfile(abs_f):
            # Store the validated absolute path
            if abs_f not in valid_files: # Avoid duplicates if input list had them
                 valid_files.append(abs_f)
            else:
                 log_warn(f"Duplicate path detected after resolving to absolute path: {abs_f}. Using only once.")
        else:
            invalid_files.append(f) # Report the original path that failed
            log_warn(f"- Not found or not a file: {abs_f} (from input '{f}')")

    log_step(f"Finished checking file existence. Found {len(valid_files)} unique, valid absolute file paths.")

    if invalid_files:
        log_warn(f"\nThe following specified paths do not exist or are not files and will be skipped:")
        for f in invalid_files:
            log_warn(f"- {f}")
        # Allow continuing if *some* files are valid? No, let's exit if any are invalid.
        log_error("\nError: One or more specified files were not found. Exiting.")
        exit(1)
    elif not valid_files:
         log_error("\nError: No valid files specified to process. Exiting.")
         exit(1)


    # --- Execute API Call ---
    log_step(f"Calling process_files_with_gemini for {len(valid_files)} valid file(s)...")
    # Get both the result message/text and the usage info
    api_result_output, usage_details = process_files_with_gemini(valid_files, user_prompt, args.inplace)
    log_step("Returned from process_files_with_gemini.")

    # --- Display Result/Status ---
    print(Fore.MAGENTA + "\n--- Gemini API Result ---")
    # If inplace, api_result_output is the status message.
    # If not inplace, api_result_output is the response text.
    print(api_result_output) # Print the main result (response text or status, may already have color)
    print(Fore.MAGENTA + "-------------------------\n")


    # --- Display Token, Cost, Duration, and Saved File Info ---
    print(Fore.MAGENTA + "\n--- Usage & Cost Estimation ---") # Keep the header
    opened_file = False # Flag to track if a file was opened
    if usage_details: # Check if usage_details dictionary exists
        # Print token/cost info if no error occurred during calculation
        if usage_details.get('error') is None or "Usage metadata missing" in usage_details.get('error', ""): # Show info even if only metadata missing
            log_info(f"Input Tokens:  {usage_details.get('input_tokens', 'N/A')}")
            log_info(f"Output Tokens: {usage_details.get('output_tokens', 'N/A')}")
            cost = usage_details.get('cost', 0.0)
            if usage_details.get('error') is None : # Only show cost success if no error at all
                 log_success(f"Estimated Cost: ${cost:.6f}")
            else: # Show cost as N/A if metadata was missing
                 log_warn(f"Estimated Cost: N/A (due to missing usage metadata)")

        # Print error if one occurred during usage/cost processing (and wasn't just missing metadata)
        elif usage_details.get('error') and "Usage metadata missing" not in usage_details.get('error', ""):
            log_error(f"Could not calculate usage/cost. Error: {usage_details['error']}")

        # Print API call duration if available
        duration = usage_details.get('api_call_duration')
        if duration is not None:
             log_info(f"API Call Duration: {duration:.3f} seconds")

        # Print saved file paths if they exist in the dictionary
        prompt_file = usage_details.get('prompt_file_path')
        raw_file = usage_details.get('raw_response_file')
        final_file = usage_details.get('cleaned_response_file')

        if prompt_file:
            log_info(f"Final prompt saved to:      {prompt_file}")
        if raw_file:
            log_info(f"Raw response saved to:      {raw_file}")
        if final_file:
            log_info(f"Final response text saved to: {final_file}")

            # --- Attempt to open the final file ---
            # Only attempt to open if NOT in inplace mode OR if inplace mode had errors (to view the problematic response)
            should_open = not args.inplace or (args.inplace and usage_details.get('error') is not None)
            # Also check if the result message indicates parsing failure or other errors
            if args.inplace and ("Fatal Error" in api_result_output or "Errors encountered" in api_result_output or "partially" in api_result_output):
                 should_open = True

            if should_open:
                log_step(f"Attempting to open final response file in browser: {final_file}")
                try:
                    # Use file:// URI scheme for local files
                    file_uri = f"file://{os.path.abspath(final_file)}"
                    webbrowser.open(file_uri)
                    log_success(f"Opened (or attempted to open) {final_file} in default browser.")
                    opened_file = True
                except Exception as wb_e:
                    log_error(f"Failed to open file in browser: {wb_e}")
                    log_warn("Please open the file manually.")
            elif args.inplace:
                 log_info("Skipping automatic opening of response file in browser (inplace successful).")
            # --- End attempt to open file ---

    else:
         log_error("Could not retrieve usage details.")

    print(Fore.MAGENTA + "-----------------------------")
    if not opened_file and usage_details and usage_details.get('cleaned_response_file'):
         # Provide manual open instruction if it wasn't opened automatically and the file exists
         final_file_path = usage_details.get('cleaned_response_file')
         if final_file_path: # Check again it exists
             log_warn(f"Browser could not be opened automatically or was skipped. Please open the final response file manually: {final_file_path}")


    log_step("Script execution finished.")

