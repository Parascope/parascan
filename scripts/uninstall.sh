#!/bin/sh

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Remove symlink from /usr/local/bin
if [ -L "/usr/local/bin/sitedog" ] || [ -f "/usr/local/bin/sitedog" ]; then
    echo "Removing /usr/local/bin/sitedog (may require sudo)..."
    sudo rm -f /usr/local/bin/sitedog
    echo "${GREEN}Removed /usr/local/bin/sitedog${NC}"
fi

# Remove ~/.sitedog directory completely
if [ -d "$HOME/.sitedog" ]; then
    rm -rf "$HOME/.sitedog"
    echo "${GREEN}Removed $HOME/.sitedog directory${NC}"
fi

# Remove PATH entries from shell profiles
for RC in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.config/fish/config.fish"; do
    if [ -f "$RC" ]; then
        # Remove lines that add sitedog to PATH
        case "$(uname)" in
            Darwin*)
                sed -i '' '/sitedog\/bin.*PATH/d' "$RC"
                ;;
            *)
                sed -i '/sitedog\/bin.*PATH/d' "$RC"
                ;;
        esac
        echo "${GREEN}Cleaned PATH from $RC${NC}"
    fi
done

echo "${GREEN}SiteDog has been fully uninstalled!${NC}" 