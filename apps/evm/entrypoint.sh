#!/bin/sh
set -e

sleep 5

# Function to extract --home value from arguments
get_home_dir() {
  home_dir="$HOME/.evm"

  # Parse arguments to find --home
  while [ $# -gt 0 ]; do
    case "$1" in
      --home)
        if [ -n "$2" ]; then
          home_dir="$2"
          break
        fi
        ;;
      --home=*)
        home_dir="${1#--home=}"
        break
        ;;
    esac
    shift
  done

  echo "$home_dir"
}

# Get the home directory (either from --home flag or default)
CONFIG_HOME=$(get_home_dir "$@")

if [ ! -f "$CONFIG_HOME/config/node_key.json" ]; then

  # Build init flags array
  init_flags="--home=$CONFIG_HOME"

  # Add required flags if environment variables are set
  if [ -n "$EVM_SIGNER_PASSPHRASE_FILE" ]; then
    init_flags="$init_flags --evnode.node.aggregator=true --evnode.signer.passphrase_file $EVM_SIGNER_PASSPHRASE_FILE"
  elif [ -n "$EVM_SIGNER_PASSPHRASE" ]; then
    init_flags="$init_flags --evnode.node.aggregator=true --evnode.signer.passphrase $EVM_SIGNER_PASSPHRASE"
  fi

  INIT_COMMAND="evm init $init_flags"
  echo "Create default config with command:"
  echo "$INIT_COMMAND"
  $INIT_COMMAND
fi


# Build start flags array
default_flags="--home=$CONFIG_HOME"

# Add required flags if environment variables are set
if [ -n "$EVM_JWT_SECRET_FILE" ]; then
  default_flags="$default_flags --evm.jwt-secret-file $EVM_JWT_SECRET_FILE"
elif [ -n "$EVM_JWT_SECRET" ]; then
  default_flags="$default_flags --evm.jwt-secret $EVM_JWT_SECRET"
fi

if [ -n "$EVM_GENESIS_HASH" ]; then
  default_flags="$default_flags --evm.genesis-hash $EVM_GENESIS_HASH"
fi

if [ -n "$EVM_ENGINE_URL" ]; then
  default_flags="$default_flags --evm.engine-url $EVM_ENGINE_URL"
fi

if [ -n "$EVM_ETH_URL" ]; then
  default_flags="$default_flags --evm.eth-url $EVM_ETH_URL"
fi

if [ -n "$EVM_BLOCK_TIME" ]; then
  default_flags="$default_flags --evnode.node.block_time $EVM_BLOCK_TIME"
fi

if [ -n "$EVM_SIGNER_PASSPHRASE_FILE" ]; then
  default_flags="$default_flags --evnode.node.aggregator=true --evnode.signer.passphrase_file $EVM_SIGNER_PASSPHRASE_FILE"
elif [ -n "$EVM_SIGNER_PASSPHRASE" ]; then
  default_flags="$default_flags --evnode.node.aggregator=true --evnode.signer.passphrase $EVM_SIGNER_PASSPHRASE"
fi

# Conditionally add DA-related flags
if [ -n "$DA_ADDRESS" ]; then
  default_flags="$default_flags --evnode.da.address $DA_ADDRESS"
fi

if [ -n "$DA_AUTH_TOKEN" ]; then
  default_flags="$default_flags --evnode.da.auth_token $DA_AUTH_TOKEN"
fi

if [ -n "$DA_NAMESPACE" ]; then
  default_flags="$default_flags --evnode.da.namespace $DA_NAMESPACE"
fi

if [ -n "$DA_DATA_NAMESPACE" ]; then
  default_flags="$default_flags --evnode.da.data_namespace $DA_DATA_NAMESPACE"
fi

if [ -n "$DA_SIGNING_ADDRESSES" ]; then
  default_flags="$default_flags --evnode.da.signing_addresses $DA_SIGNING_ADDRESSES"
fi

if [ -n "$DA_BLOCK_TIME" ]; then
  default_flags="$default_flags --evnode.da.block_time $DA_BLOCK_TIME"
fi

# If no arguments passed, show help
if [ $# -eq 0 ]; then
  exec evm
fi

# If first argument is "start", apply default flags
if [ "$1" = "start" ]; then
  shift
  START_COMMAND="evm start $default_flags"
  echo "Create default config with command:"
  echo "$START_COMMAND \"$@\""
  exec $START_COMMAND "$@"

else
  # For any other command/subcommand, pass through directly
  exec evm "$@"
fi
