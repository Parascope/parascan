#!/bin/sh

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Remove symlink from /usr/local/bin
if [ -L "/usr/local/bin/para" ] || [ -f "/usr/local/bin/para" ]; then
    echo "Removing /usr/local/bin/para (may require sudo)..."
    sudo rm -f /usr/local/bin/para
    echo "${GREEN}Removed /usr/local/bin/para${NC}"
fi

# Remove ~/.parascope directory completely
if [ -d "$HOME/.parascope" ]; then
    rm -rf "$HOME/.parascope"
    echo "${GREEN}Removed $HOME/.parascope directory${NC}"
fi

# Remove PATH entries from shell profiles
for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.config/fish/config.fish"; do
    if [ -f "$RC" ]; then
        # Remove lines that add parascope to PATH
        case "$(uname)" in
            Darwin*)
                sed -i '' '/parascope\/bin.*PATH/d' "$RC"
                ;;
            *)
                sed -i '/parascope\/bin.*PATH/d' "$RC"
                ;;
        esac
        echo "${GREEN}Cleaned PATH from $RC${NC}"
    fi
done

echo "${GREEN}Parascan has been fully uninstalled!${NC}"