import google.generativeai as genai
import google.auth
import os
import argparse
import re
import tempfile
from typing import Tuple, Dict, Any
import colorama
from colorama import Fore, Style, init
import time
import webbrowser
import markdown
import codecs

MODEL_NAME_PRO = "gemini-2.5-pro-preview-05-06"
MODEL_NAME_FLASH = "gemini-2.5-flash-preview-05-20" # New Flash model

DEFAULT_SOURCE_DIR = "/usr/local/google/home/zicong/code/src/user/zicong/cloudrun-controller/internal/controller/"
DEFAULT_SOURCE_FILES = [
    DEFAULT_SOURCE_DIR + "service_controller.go",
    DEFAULT_SOURCE_DIR + "service.go",
    DEFAULT_SOURCE_DIR + "utils.go",
]

# Pricing for gemini-2.5-pro-preview-05-06
TOKEN_LIMIT_TIER_1_PRO = 200000 # Tokens
PRICE_INPUT_TIER_1_PRO = 1.25 # Prompts <= 200k tokens
PRICE_INPUT_TIER_2_PRO = 2.50 # Prompts > 200k tokens
PRICE_OUTPUT_TIER_1_PRO = 10.00 # Prompts <= 200k tokens
PRICE_OUTPUT_TIER_2_PRO = 15.00 # Prompts > 200k tokens

# Pricing for gemini-2.5-flash-preview-05-20 (per 1M tokens)
PRICE_INPUT_FLASH_NON_THINKING = 0.60 # Non-thinking
PRICE_OUTPUT_FLASH_THINKING = 3.50 # Thinking
PRICE_INPUT_FLASH = 0.15 # Input (based on user provided price)

def log_info(message):
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.CYAN}[{timestamp} INFO]{Style.RESET_ALL} {message}")

def log_step(message):
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.BLUE}[{timestamp} STEP]{Style.RESET_ALL} {message}")

def log_warn(message):
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.YELLOW}[{timestamp} WARN]{Style.RESET_ALL} {message}")

def log_error(message):
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.RED}[{timestamp} ERROR]{Style.RESET_ALL} {message}")

def log_success(message):
    timestamp = time.strftime("%H:%M:%S", time.localtime())
    print(f"{Fore.GREEN}[{timestamp} OK]{Style.RESET_ALL} {message}")


def process_files_with_gemini(file_paths: list[str], prompt: str, inplace: bool, use_flash_model: bool) -> Tuple[str, Dict[str, Any]]:
    """
    Reads content from EXISTING files, sends it to Gemini,
    saves responses, calculates usage/cost, and either returns the
    response or attempts to modify files inplace based on the text format.

    Args:
        file_paths: A list of paths to EXISTING source files.
        prompt: The user's question or instruction for the API. MUST NOT be empty.
        inplace: If True, request text format and attempt to modify files inplace.
        use_flash_model: If True, use the Flash model and its pricing.

    Returns:
        A tuple containing:
        - The text response or status message from the Gemini API (str).
        - A dictionary with usage info.
    """
    log_step(f"Entering process_files_with_gemini for {len(file_paths)} files.")
    if not prompt:
        err_msg = "Error: The prompt provided to process_files_with_gemini cannot be empty."
        log_error(err_msg)
        return err_msg, {
            'input_tokens': 0, 'output_tokens': 0, 'cost': 0.0,
            'api_call_duration': None, 'prompt_file_path': None,
            'raw_response_file': None, 'cleaned_response_file': None,
            'html_result_file': None, 'error': "Empty prompt provided."
        }

    current_model_name = MODEL_NAME_FLASH if use_flash_model else MODEL_NAME_PRO
    log_info(f"Using model: {current_model_name}")

    files_content_map = {}
    combined_content_for_prompt = ""
    usage_info = {
        'input_tokens': 0,
        'output_tokens': 0,
        'cost': 0.0,
        'api_call_duration': None,
        'prompt_file_path': None,
        'raw_response_file': None,
        'cleaned_response_file': None,
        'html_result_file': None,
        'error': None
    }
    log_step("Starting file reading loop...")
    for file_path in file_paths:
        try:
            abs_file_path = os.path.abspath(file_path)
            if not os.path.exists(abs_file_path):
                 log_warn(f"File path provided but not found - {abs_file_path}. Skipping.")
                 continue
            log_info(f"Reading file: {abs_file_path}")
            with open(abs_file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                if abs_file_path in files_content_map:
                    log_warn(f"Duplicate absolute file path '{abs_file_path}' detected in input. Using content from the last occurrence.")
                files_content_map[abs_file_path] = content
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

    log_step("Constructing final prompt...")
    final_prompt = f"{prompt}\n\n{combined_content_for_prompt}"

    if inplace:
        text_format_instruction = (
            "\n\nIMPORTANT: Respond ONLY with the complete, modified content for each file, "
            "formatted exactly as follows, using the ABSOLUTE file paths provided:\n"
        )
        absolute_paths_in_prompt = list(files_content_map.keys())
        for abs_path in absolute_paths_in_prompt:
             text_format_instruction += (
                 f"--- BEGIN of {abs_path} ---\n"
                 f"{{content for {abs_path}}}\n"
                 f"--- END of {abs_path} ---\n"
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

    log_step("Attempting to save final prompt to /tmp...")
    try:
        with tempfile.NamedTemporaryFile(
            mode='w', encoding='utf-8',
            prefix='gemini_final_prompt_', suffix='.txt',
            dir='/tmp', delete=False
        ) as tmp_prompt_file:
            tmp_prompt_file.write(final_prompt)
            prompt_file_path = tmp_prompt_file.name
            usage_info['prompt_file_path'] = prompt_file_path
            log_info(f"Saved final prompt to: {prompt_file_path}")
    except Exception as save_prompt_e:
        log_warn(f"Failed to save final prompt to /tmp. Error: {save_prompt_e}")
    log_step("Finished saving final prompt.")


    raw_response_text = ""
    response = None
    api_duration = None
    try:
        model = genai.GenerativeModel(current_model_name)

        log_step("Estimating input tokens (client-side)...")
        try:
            count_response = model.count_tokens(final_prompt)
            input_token_estimate = count_response.total_tokens
            log_info(f"Estimated input tokens (client-side count): {input_token_estimate}")
        except Exception as count_e:
            log_warn(f"Could not estimate input tokens client-side: {count_e}")

        log_step(f"Sending prompt to Gemini model: {current_model_name}...")
        start_time = time.monotonic()
        response = model.generate_content(final_prompt)
        end_time = time.monotonic()
        api_duration = end_time - start_time
        usage_info['api_call_duration'] = api_duration

        raw_response_text = response.text
        log_success(f"Received response from Gemini. Duration: {api_duration:.3f} seconds")


        log_step("Processing usage metadata...")
        input_tokens = 0
        output_tokens = 0
        if response and hasattr(response, 'usage_metadata'):
            input_tokens = response.usage_metadata.prompt_token_count
            output_tokens = response.usage_metadata.candidates_token_count
            usage_info['input_tokens'] = input_tokens
            usage_info['output_tokens'] = output_tokens
            log_info(f"API Usage: Input Tokens = {input_tokens}, Output Tokens = {output_tokens}")

            log_step("Calculating estimated cost...")
            if use_flash_model:
                log_info(f"Using Flash model pricing.")
                # Flash model pricing:
                # Input (prompt): $0.15 per 1M tokens
                # Output (non-thinking / grounded generation / embeddings): $0.60 per 1M tokens
                # Output (thinking / non-grounded generation / chat): $3.50 per 1M tokens
                # For simplicity, we'll use the provided "input" rate for input tokens,
                # and assume "thinking" rate for output tokens as that's the common use case for generation.
                input_price = PRICE_INPUT_FLASH
                output_price = PRICE_OUTPUT_FLASH_THINKING
                log_info(f"Flash pricing: Input=${input_price}/1M, Output (Thinking)=${output_price}/1M")
            else: # Pro model pricing
                log_info(f"Using Pro model pricing.")
                if input_tokens > TOKEN_LIMIT_TIER_1_PRO:
                    input_price = PRICE_INPUT_TIER_2_PRO
                    output_price = PRICE_OUTPUT_TIER_2_PRO
                    log_info(f"Input tokens ({input_tokens}) > {TOKEN_LIMIT_TIER_1_PRO}, using Tier 2 pricing for Pro.")
                else:
                    input_price = PRICE_INPUT_TIER_1_PRO
                    output_price = PRICE_OUTPUT_TIER_1_PRO
                    log_info(f"Input tokens ({input_tokens}) <= {TOKEN_LIMIT_TIER_1_PRO}, using Tier 1 pricing for Pro.")

            input_cost = (input_tokens / 1_000_000) * input_price
            output_cost = (output_tokens / 1_000_000) * output_price
            total_cost = input_cost + output_cost
            usage_info['cost'] = total_cost
            log_info(f"Calculated Cost: Input=${input_cost:.6f}, Output=${output_cost:.6f}, Total=${total_cost:.6f}")
            log_step("Finished cost calculation.")

        else:
            log_warn("Usage metadata not found in response. Cannot calculate exact tokens or cost.")
            usage_info['error'] = "Usage metadata missing."
        log_step("Finished processing usage metadata.")


        log_step("Attempting to save raw response...")
        try:
            with tempfile.NamedTemporaryFile(
                mode='w', encoding='utf-8',
                prefix='gemini_raw_response_', suffix='.txt',
                dir='/tmp', delete=False
            ) as tmp_file:
                tmp_file.write(raw_response_text)
                raw_response_path = tmp_file.name
                usage_info['raw_response_file'] = raw_response_path
                log_info(f"Saved raw Gemini response to: {raw_response_path}")
        except Exception as save_e:
            log_warn(f"Failed to save raw Gemini response to /tmp. Error: {save_e}")
        log_step("Finished saving raw response.")


        log_step("Cleaning response text (if not inplace)...")
        response_text = raw_response_text
        if not inplace:
            response_text = raw_response_text.strip()
            cleaned = False
            if response_text.startswith("```") and response_text.endswith("```"):
                 first_newline = response_text.find('\n')
                 if first_newline != -1:
                     potential_lang = response_text[3:first_newline].strip()
                     if potential_lang and not potential_lang.startswith('-') and ' ' not in potential_lang:
                         response_text = response_text[first_newline+1:-3].strip()
                         cleaned = True
                     else:
                         response_text = response_text[3:-3].strip()
                         cleaned = True
                 else:
                     response_text = response_text[3:-3].strip()
                     cleaned = True

            if cleaned:
                log_info("Removed potential markdown code fences from response (for non-inplace).")
        else:
            log_info("Skipping general cleaning for inplace mode (expecting specific absolute path format).")
        log_step("Finished cleaning response text.")

        log_step("Attempting to save final response text...")
        try:
            with tempfile.NamedTemporaryFile(
                mode='w', encoding='utf-8',
                prefix='gemini_final_response_', suffix='.txt',
                dir='/tmp', delete=False
            ) as tmp_file:
                tmp_file.write(response_text)
                cleaned_response_path = tmp_file.name
                usage_info['cleaned_response_file'] = cleaned_response_path
                log_info(f"Saved final Gemini response text to: {cleaned_response_path}")
        except Exception as save_e:
            log_warn(f"Failed to save final Gemini response text to /tmp. Error: {save_e}")
        log_step("Finished saving final response text.")

    except Exception as e:
        error_msg = f"An error occurred while calling the Gemini API: {e}"
        log_error(error_msg)
        usage_info['error'] = str(e)
        if api_duration is not None:
             usage_info['api_call_duration'] = api_duration
        return f"Error: API call failed - {e}", usage_info

    if inplace:
        log_step("Starting inplace modification process (parsing absolute path text format)...")
        modified_files_count = 0
        errors = []
        extracted_content_map: Dict[str, str] = {}

        if not response_text:
             err_msg = "Fatal Error: Received empty response from Gemini. Cannot parse text format."
             log_error(err_msg)
             errors.append(err_msg)
        else:
            log_step("Attempting to parse text response using BEGIN/END markers (expecting absolute paths)...")
            pattern = re.compile(r"--- BEGIN of (.*?) ---\s*(.*?)\s*--- END of \1 ---", re.DOTALL)
            matches = pattern.finditer(response_text)

            found_matches = False
            for match in matches:
                found_matches = True
                abs_path_from_response = match.group(1).strip()
                new_content = match.group(2).strip()

                if abs_path_from_response not in files_content_map:
                     log_warn(f"Response included block for path '{abs_path_from_response}', which was NOT in the original input files. Ignoring this block.")
                     errors.append(f"Extraneous content block for '{abs_path_from_response}' in response.")
                     continue

                if abs_path_from_response in extracted_content_map:
                    log_warn(f"Duplicate BEGIN/END block found for absolute path '{abs_path_from_response}' in response. Using the last occurrence.")
                extracted_content_map[abs_path_from_response] = new_content
                log_info(f"Extracted content for absolute path: '{abs_path_from_response}'")

            if not found_matches:
                 err_msg = "Fatal Error: Could not find any valid '--- BEGIN of ... --- ... --- END of ... ---' blocks in the response."
                 log_error(err_msg)
                 errors.append(err_msg)
            elif not extracted_content_map and found_matches:
                 err_msg = "Fatal Error: Found BEGIN/END blocks, but none matched the absolute paths of the input files."
                 log_error(err_msg)
                 errors.append(err_msg)
            elif extracted_content_map:
                 log_success(f"Successfully parsed text response, found valid content for {len(extracted_content_map)} file(s).")

            log_step("Finished text parsing attempt.")


        if not errors and extracted_content_map:
            log_step("Starting file overwrite loop (using validated absolute paths)...")
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
                    log_warn(f"No valid '--- BEGIN/END ---' block found in response for absolute path '{abs_path}'. File not changed.")
                    errors.append(f"Missing content block for '{abs_path}' in response.")
            log_step("Finished file overwrite loop.")

        log_step("Compiling inplace modification results...")
        if not errors and modified_files_count == len(files_content_map):
             result_message = f"Inplace modification complete. {modified_files_count} file(s) modified based on text response (using absolute paths)."
             log_success(result_message)
        elif not errors and modified_files_count < len(files_content_map):
             result_message = f"Inplace modification attempted. {modified_files_count} out of {len(files_content_map)} file(s) modified."
             result_message += "\nErrors/Warnings (check logs for details):\n"
             missing_count = len(files_content_map) - modified_files_count
             result_message += f"- {missing_count} file(s) were not modified (missing content blocks in response).\n"
             result_message += "\n".join([f"- {e}" for e in errors if "Missing content block" not in e])
             log_warn("Inplace modification finished partially (some files not modified).")
        else:
             result_message = f"Inplace modification attempted. {modified_files_count} file(s) modified."
             result_message += "\nErrors encountered during processing/writing:\n" + "\n".join([f"- {e}" for e in errors])
             if not extracted_content_map and any("Fatal Error" in err for err in errors):
                 result_message += "\n\n" + Fore.RED + "Text parsing failed or yielded no valid content. No files were modified."
             log_warn("Inplace modification finished with errors.")

        usage_info['error'] = "; ".join(errors) if errors else None

        return result_message, usage_info

    else:
        log_step("Exiting process_files_with_gemini (inplace=False).")
        return response_text, usage_info

if __name__ == "__main__":
    init(autoreset=True)
    log_step("Script execution started.")

    parser = argparse.ArgumentParser(description="Process EXISTING files with the Gemini API, calculate cost, optionally modify them inplace (text format), and convert final non-inplace response to HTML.")

    log_step("Parsing command line arguments...")
    parser.add_argument(
        "files",
        nargs='*',
        default=None,
        help="Paths to the source files to process directly. Use --file-list instead to read paths from a file."
    )
    parser.add_argument(
        "--file-list",
        type=str,
        default=None,
        help="Path to a file containing a list of source file paths (one per line). Overrides any files provided directly."
    )
    parser.add_argument(
        "--inplace",
        action="store_true",
        help="Modify the source files inplace based on Gemini text response format (using absolute paths). **USE WITH CAUTION**"
    )
    parser.add_argument(
        "--prompt",
        type=str,
        required=True,
        help="The base prompt/instruction for the Gemini API (REQUIRED). Format instructions are added automatically for --inplace."
    )
    parser.add_argument(
        "--api-key",
        type=str,
        default=None,
        help="Your Gemini API key. If not provided, attempts to use gcloud default credentials."
    )
    parser.add_argument(
        "--flash",
        action="store_true",
        help=f"Use the {MODEL_NAME_FLASH} model instead of {MODEL_NAME_PRO}. Pricing: Input (Prompt) ${PRICE_INPUT_FLASH}/1M, Output (Non-Thinking) ${PRICE_INPUT_FLASH_NON_THINKING}/1M, Output (Thinking) ${PRICE_OUTPUT_FLASH_THINKING}/1M tokens."
    )
    args = parser.parse_args()
    log_step("Finished parsing arguments.")
    log_info(f"Arguments: files={args.files}, file_list={args.file_list}, inplace={args.inplace}, flash={args.flash}, prompt='{args.prompt[:50]}...', api_key={'SET' if args.api_key else 'NOT SET'}")

    log_step("Attempting Authentication...")
    if args.api_key:
        try:
            genai.configure(api_key=args.api_key)
            log_success("Successfully authenticated using provided API key.")
        except Exception as e:
            log_error(f"Failed to configure Gemini with API key: {e}")
            exit(1)
    else:
        try:
            credentials, project_id = google.auth.default()
            genai.configure(credentials=credentials)
            log_success(f"Successfully authenticated using ADC for project: {project_id}")
        except google.auth.exceptions.DefaultCredentialsError as e:
            log_error(f"Authentication failed. Please provide an API key using --api-key or run 'gcloud auth application-default login'. Error: {e}")
            exit(1)
        except Exception as e:
            log_error(f"An unexpected error occurred during authentication: {e}")
            exit(1)


    log_step("Determining source files to process...")
    source_files_to_check = []
    if args.file_list:
        log_info(f"Reading file list from: {args.file_list}")
        try:
            with open(args.file_list, 'r', encoding='utf-8') as f_list:
                for line in f_list:
                    stripped_line = line.strip()
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

    elif args.files:
        source_files_to_check = args.files
        log_info(f"Using {len(source_files_to_check)} files provided directly via command line.")
    else:
        source_files_to_check = DEFAULT_SOURCE_FILES
        log_info(f"No files provided, using default paths: {', '.join(source_files_to_check)}")
    log_step("Finished determining source files.")

    user_prompt = args.prompt
    if not user_prompt.strip():
        log_error("Error: The --prompt argument was provided but is empty or contains only whitespace. Exiting.")
        exit(1)

    if args.inplace:
         log_warn("\n--- Running in INPLACE mode ---")
         log_warn("WARNING: This will attempt to overwrite files based on Gemini's text output (using absolute paths).")
         log_warn("Ensure your prompt clearly asks for the desired modifications in the specified format.")
         log_warn("BACK UP YOUR FILES before proceeding if you haven't already.")
         log_warn("-------------------------------\n")

    valid_files = []
    invalid_files = []
    log_step("Checking file existence...")
    for f_path in source_files_to_check:
        abs_f = os.path.abspath(f_path)
        if os.path.exists(abs_f) and os.path.isfile(abs_f):
            if abs_f not in valid_files:
                 valid_files.append(abs_f)
            else:
                 log_warn(f"Duplicate path detected after resolving to absolute path: {abs_f}. Using only once.")
        else:
            invalid_files.append(f_path)
            log_warn(f"- Not found or not a file: {abs_f} (from input '{f_path}')")

    log_step(f"Finished checking file existence. Found {len(valid_files)} unique, valid absolute file paths.")

    if invalid_files:
        log_warn(f"\nThe following specified paths do not exist or are not files and will be skipped:")
        for f_path in invalid_files:
            log_warn(f"- {f_path}")
        log_error("\nError: One or more specified files were not found. Exiting.")
        exit(1)
    elif not valid_files:
         log_error("\nError: No valid files specified to process. Exiting.")
         exit(1)


    log_step(f"Calling process_files_with_gemini for {len(valid_files)} valid file(s)...")
    api_result_output, usage_details = process_files_with_gemini(valid_files, user_prompt, args.inplace, args.flash)
    log_step("Returned from process_files_with_gemini.")

    print(Fore.MAGENTA + "\n--- Gemini API Result ---")
    print(api_result_output)
    print(Fore.MAGENTA + "-------------------------\n")


    print(Fore.MAGENTA + "\n--- Usage, Cost & Output Files ---")
    opened_file = False
    html_file_path = None

    if usage_details:
        usage_error = usage_details.get('error')
        show_cost_info = not usage_error or "Usage metadata missing" in usage_error or args.inplace or "Empty prompt provided" in usage_error

        if show_cost_info:
            input_tokens_str = usage_details.get('input_tokens', 'N/A')
            output_tokens_str = usage_details.get('output_tokens', 'N/A')
            cost_str = f"${usage_details.get('cost', 0.0):.6f}" if usage_details.get('cost') is not None else "N/A"

            if "Empty prompt provided" in (usage_error or ""):
                log_warn("Input Tokens:  N/A (API call skipped due to empty prompt)")
                log_warn("Output Tokens: N/A (API call skipped due to empty prompt)")
                log_warn("Estimated Cost: N/A (API call skipped due to empty prompt)")
            else:
                log_info(f"Input Tokens:  {input_tokens_str}")
                log_info(f"Output Tokens: {output_tokens_str}")

                if "Usage metadata missing" in (usage_error or ""):
                    log_warn(f"Estimated Cost: N/A (due to missing usage metadata)")
                elif usage_details.get('cost') is not None:
                    log_success(f"Estimated Cost: {cost_str}")
                else:
                    log_warn(f"Estimated Cost: N/A")

        elif usage_error:
            log_error(f"Could not calculate usage/cost. Error: {usage_error}")

        duration = usage_details.get('api_call_duration')
        if duration is not None:
             log_info(f"API Call Duration: {duration:.3f} seconds")
        elif "Empty prompt provided" in (usage_error or ""):
             log_info("API Call Duration: N/A (API call skipped)")


        prompt_file = usage_details.get('prompt_file_path')
        raw_file = usage_details.get('raw_response_file')
        final_file = usage_details.get('cleaned_response_file')

        if prompt_file:
            log_info(f"Final prompt saved to:      {prompt_file}")
        if raw_file:
            log_info(f"Raw response saved to:      {raw_file}")
        if final_file:
            log_info(f"Final response text saved to: {final_file}")

            log_step(f"Attempting to convert final response '{final_file}' to HTML...")
            try:
                with open(final_file, 'r', encoding='utf-8') as md_file:
                    markdown_content = md_file.read()

                html_head = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gemini API Result</title>
    <style>
        body { font-family: sans-serif; line-height: 1.6; padding: 20px; max-width: 900px; margin: auto; color: #333; }
        pre { background-color: #f4f4f4; padding: 1em; border-radius: 5px; overflow-x: auto; border: 1px solid #ddd; }
        code { font-family: Consolas, 'Courier New', monospace; background-color: #f4f4f4; padding: 0.2em 0.4em; border-radius: 3px;}
        pre > code { background-color: transparent; padding: 0; border-radius: 0; border: none; }
        table { border-collapse: collapse; width: 100%; margin-bottom: 1em; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; font-weight: bold; }
        blockquote { border-left: 4px solid #ccc; padding-left: 1em; margin-left: 0; color: #555; font-style: italic; }
        h1, h2, h3, h4, h5, h6 { border-bottom: 1px solid #eee; padding-bottom: 0.3em; margin-top: 1.5em; }
        img { max-width: 100%; height: auto; }
        ul, ol { padding-left: 2em; }
    </style>
</head>
<body>
<h1>Gemini API Result</h1>
"""
                html_body = markdown.markdown(markdown_content, extensions=['fenced_code', 'tables', 'sane_lists'])
                html_foot = "</body>\n</html>"
                full_html = html_head + html_body + html_foot

                with tempfile.NamedTemporaryFile(
                    mode='w', encoding='utf-8',
                    prefix='gemini_html_result_', suffix=".html",
                    dir='/tmp', delete=False
                ) as tmp_html_file:
                    with codecs.open(tmp_html_file.name, 'w', encoding='utf-8') as f_html:
                         f_html.write(full_html)
                    html_file_path = tmp_html_file.name

                log_success(f"Successfully converted response to HTML: {html_file_path}")
                usage_details['html_result_file'] = html_file_path

            except ImportError:
                 log_error("The 'markdown' library is required for HTML conversion but not installed.")
                 log_warn("Please install it: pip install markdown")
            except Exception as html_e:
                log_error(f"Failed to convert response to HTML: {html_e}")


        file_to_open = html_file_path if html_file_path else final_file
        open_target_type = "HTML result file" if html_file_path else "final response text file"

        should_open = False
        if file_to_open and "Empty prompt provided" not in (usage_error or ""):
            if not args.inplace:
                 should_open = True
            elif args.inplace:
                 inplace_had_errors = (
                     usage_details.get('error') is not None or
                     (api_result_output and ("Fatal Error" in api_result_output or "Errors encountered" in api_result_output or "partially" in api_result_output or "Missing content block" in api_result_output))
                 )
                 if inplace_had_errors:
                      should_open = True

        if should_open:
            log_step(f"Attempting to open {open_target_type} in browser: {file_to_open}")
            try:
                file_uri = f"file://{os.path.abspath(file_to_open)}"
                webbrowser.open(file_uri)
                log_success(f"Opened (or attempted to open) {file_to_open} in default browser.")
                opened_file = True
            except Exception as wb_e:
                log_error(f"Failed to open {open_target_type} in browser: {wb_e}")
        elif args.inplace and not should_open and "Empty prompt provided" not in (usage_error or ""):
             log_info("Skipping automatic opening of response file in browser (inplace successful or no errors detected).")

    else:
         log_error("Could not retrieve usage details.")


    print(Fore.MAGENTA + "------------------------------------")
    if not opened_file and usage_details:
         manual_open_path = usage_details.get('html_result_file')
         if not manual_open_path:
             manual_open_path = usage_details.get('cleaned_response_file')

         if manual_open_path:
             log_warn(f"Browser could not be opened automatically or was skipped. Please open the result file manually: {manual_open_path}")


    log_step("Script execution finished.")


try:
    import markdown
except ImportError:
    init(autoreset=True)
    log_error("Fatal Error: The required 'markdown' library is not installed.")
    log_warn("Please install it using pip: pip install markdown")
    exit(1)