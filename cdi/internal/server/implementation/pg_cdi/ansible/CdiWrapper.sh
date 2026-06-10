#!/bin/bash

# CDI path
CDI_PATH="/home/cdiadmin/bin/"

# CDIv1.1 path
CDIv1_1=$CDI_PATH"cdictl"
# CDIv1.1 value of Fablic_Idx
FABLIC_IDX=1
# CDIv1.1 login user
CDIv1_1_USER_EXPECT="Enter username: "
# CDIv1.1 login password
CDIv1_1_PASS_EXPECT="Password for"

# CDIv1.0 login user
CDIv1_0_USER_EXPECT="Enter Your Username : "
# CDIv1.0 login password
CDIv1_0_PASS_EXPECT="Enter Your Password : "

while getopts "u:p:g:" opt; do
  case $opt in
    u)  # login user  
      login_user="$OPTARG"
      ;;
    p)  # login password
      login_password="$OPTARG"
      ;;
    g)  # CDI guiest
      cdi_guest="$OPTARG"
      ;;
    \?)
      echo "Invalid argument: -$OPTARG" >&2
      exit 1
      ;;
    :)
      echo "option -$OPTARG is required argument" >&2
      exit 1
      ;;
  esac
done
shift $((OPTIND-1))

# command
cmd="$1 $2"

# generate command line
for N in "${@}" ;
do
  cmdline+="$N "
done

# judge for CDI v1.0 or v1.1 
if [ -e $CDIv1_1 ]; then
    # configure for login
    URL="https://$cdi_guest/"
    login_path=$CDI_PATH
    CDIv1_1_PASS_EXPECT="$CDIv1_1_PASS_EXPECT $login_user: "
    login_user_expect=$CDIv1_1_USER_EXPECT
    login_pass_expect=$CDIv1_1_PASS_EXPECT

    # generate v1.1 command
    login="./cdictl -u $URL auth login"
    cmdline="./cdictl $cmdline -f $FABLIC_IDX"

else
    # configure for login
    URL="https://$cdi_guest/resource_manager/api/v1"
    login_path=$CDI_PATH
    login_user_expect=$CDIv1_0_USER_EXPECT
    login_pass_expect=$CDIv1_0_PASS_EXPECT

    # generate v1.0 command
    login="./epcctl -u $URL login"
    cmdline="./epcctl -u $URL $cmdline"
    case $cmd in
        "machine destroy"|"machine show"|"machine power"|"machine update_status"|"machine modify"|"machine p2p")
	    cmdline=$(echo "$cmdline" | sed 's/-g [^ ]*//g')	# exclusive -g option
            ;;
    esac
fi

# login and execution
cd $login_path
output=$(expect -c "
set timeout 60
spawn $login
expect \"$login_user_expect\"
send \"$login_user\n\"
expect \"$login_pass_expect\"
send \"$login_password\n\"
expect \"cdi:\"
send \"$cmdline\n\"
expect \"cdi:\"
")
expect_exit_code=$?

# Handle expect command errors before parsing
if [ $expect_exit_code -ne 0 ] || ! echo "$output" | grep -q "cdi:[[:space:]]*$"; then
    if [ -e $CDIv1_1 ]; then
        echo "RESULT_TYPE:ERROR_V_1_1"
    else
        echo "RESULT_TYPE:ERROR_V_1_0"
    fi
    echo "CDI login failed."
    exit 1
fi

# Function to parse and format CDI(v1.0) output for consistent processing
parse_and_format_output_v1_0() {
    local output="$1"
    
    # Check for success code pattern (2xx) - look for 200, 201, 202, etc. as standalone lines
    if echo "$output" | grep -qE '(^|[[:space:]])2[0-9][0-9]([[:space:]]|$)'; then
        echo "RESULT_TYPE:SUCCESS"
        
        # Extract data after the 2xx code and before final "cdi:"
        local result_data=$(echo "$output" | sed -n '/^[[:space:]]*2[0-9][0-9][[:space:]]*$/,/cdi:[[:space:]]*$/p' | sed '1d;$d')
        
        # Clean up the result data
        result_data=$(echo "$result_data" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        if [ -n "$result_data" ]; then
            echo "$result_data" | python3 -c "
import json, ast, sys
raw = sys.stdin.read().strip()
try:
    print(json.dumps(ast.literal_eval(raw)))
except Exception:
    print(raw)
" 2>/dev/null || echo "$result_data"
        fi
        return 0
        
    # Check for error patterns
    elif echo "$output" | grep -q 'error:'; then
        echo "RESULT_TYPE:ERROR_V_1_0"
        
        # Extract error message after "error:"
        local error_msg=$(echo "$output" | grep 'error:' | sed 's/.*error: *//' | head -1)
        error_msg=$(echo "$error_msg" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        if [ -n "$error_msg" ]; then
            echo "$error_msg"
        else
            echo "Unknown error"
        fi
        return 1
        
    # Check for error code patterns (any 3-digit code that's not 2xx)
    elif echo "$output" | grep -qE '(^|[[:space:]])[0-9]{3}([[:space:]]|$)' && ! echo "$output" | grep -qE '(^|[[:space:]])2[0-9][0-9]([[:space:]]|$)'; then
        echo "RESULT_TYPE:ERROR_V_1_0"
        
        # Extract the error code and any following error message
        local error_code=$(echo "$output" | grep -oE '(^|[[:space:]])[0-9]{3}([[:space:]]|$)' | grep -vE '2[0-9][0-9]' | head -1 | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # Try to extract error message - first try with cdi: delimiter
        local error_msg=$(echo "$output" | sed -n "/${error_code}/,/cdi:[[:space:]]*$/p" | sed '1d;$d' | head -1)
        
        # If no message found with cdi: delimiter, try to get the line after the error code
        if [ -z "$error_msg" ]; then
            error_msg=$(echo "$output" | sed -n "/${error_code}/{n;p;}" | head -1)
        fi
        
        # Clean up the error message and handle special characters
        error_msg=$(echo "$error_msg" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        if [ -n "$error_msg" ]; then
            echo "${error_code} ${error_msg}"
        else
            echo "${error_code} Unknown error"
        fi
        return 1
    fi
    
    # Unknown format
    echo "RESULT_TYPE:UNKNOWN"
    echo "Unknown response format"
    return 1
}

# Function to parse and format CDI(v1.1) output for consistent processing
parse_and_format_output_v1_1() {
    local output="$1"
    
    # First, try to find "Success" pattern using string matching (more robust)
    if echo "$output" | grep -q -e "Success" -e "Accept"; then
        echo "RESULT_TYPE:SUCCESS"
        
        # Extract data after "Success" and before final "cdi:"
        # Use sed to find the section between Success and the final cdi:
        local result_data=$(echo "$output" | sed -n '/Success/,/cdi:[[:space:]]*$/p' | sed '1d;$d')
        
        # Clean up the result data
        result_data=$(echo "$result_data" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        if [ -n "$result_data" ]; then
            echo "$result_data" | python3 -c "
import json, ast, sys
raw = sys.stdin.read().strip()
try:
    print(json.dumps(ast.literal_eval(raw)))
except Exception:
    print(raw)
" 2>/dev/null || echo "$result_data"
        fi
        return 0
        
    elif echo "$output" | grep -q "Error"; then
        echo "RESULT_TYPE:ERROR_V_1_1"
        
        # Extract error message after "Error" 
        local error_msg=$(echo "$output" | sed -n '/Error/,/cdi:[[:space:]]*$/p' | sed '1d;$d' | head -1)
        error_msg=$(echo "$error_msg" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # Check if error message starts with error code pattern (E + 6 digits)
        if [[ "$error_msg" =~ ^E[0-9]{6} ]]; then
            # Extract error code and message separately
            local error_code=$(echo "$error_msg" | grep -oE '^E[0-9]{6}')
            local msg_part=$(echo "$error_msg" | sed "s/^${error_code}[[:space:]]*//" )
            echo "${error_code} ${msg_part}"
        else
            if [ -n "$error_msg" ]; then
                echo "$error_msg"
            else
                echo "Unknown error"
            fi
        fi
        return 1
    fi
    
    # Unknown format
    echo "RESULT_TYPE:UNKNOWN"
    echo "Unknown response format"
    return 1
}

# Process the output
if [ -e $CDIv1_1 ]; then
    parse_and_format_output_v1_1 "$output"
else
    parse_and_format_output_v1_0 "$output"
fi

# logout
exit

