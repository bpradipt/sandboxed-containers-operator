#!/bin/bash
# Contains common functions used by the scripts

set -x

# Defaults for pause image
# This pause image is multi-arch
PAUSE_IMAGE_REPO_DEFAULT="quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256"
PAUSE_IMAGE_VERSION_DEFAULT="7f3cb6f9d265291b47a7491c2ba4f4dd0752a18b661eee40584f9a5dbcbe13bb"
PAUSE_IMAGE_REPO_AUTH_FILE="/tmp/regauth/auth.json"

# function to trap errors and exit
function error_exit() {
    echo "$1" 1>&2
    exit 1
}

# function to install required rpm packages.
# the packages are listed in the variable REQUIRED_RPM_PACKAGES

function install_rpm_packages() {
    # Install required rpm packages
    # If any error occurs, exit the script with an error message

    # List of packages to be installed
    REQUIRED_RPM_PACKAGES=(
        "curl"
        "git"
        "make"
        "unzip"
        "skopeo"
    )

    # Create a new array to store rpm packages that are not installed
    NEW_REQUIRED_RPM_PACKAGES=()

    # Check which rpm packages are already installed and remove them from the list
    for package in "${REQUIRED_RPM_PACKAGES[@]}"; do
        if rpm -q "${package}" &>/dev/null; then
            echo "Package ${package} is already installed. Skipping."
        else
            # Add the rpm package to the new array if it's not installed
            NEW_REQUIRED_RPM_PACKAGES+=("$package")
        fi
    done

    # Update the original array with the new list of rpm packages
    REQUIRED_RPM_PACKAGES=("${NEW_REQUIRED_RPM_PACKAGES[@]}")

    # Install the required rpm packages
    if [[ "${#REQUIRED_RPM_PACKAGES[@]}" -gt 0 ]]; then
        echo "Installing required packages..."
        # Using allowerasing flag to remove conflicting packages
        # eg curl and curl-minimal
        yum install -y --allowerasing "${REQUIRED_RPM_PACKAGES[@]}" ||
            error_exit "Failed to install required packages"
    else
        echo "All required packages are already installed."
    fi

}

# function to download and install binary packages.
# the packages, their respective download locations and compression
# are available in the variable REQUIRED_BINARY_PACKAGES
# the function will download the packages, extract them and install them in /usr/local/bin
# Following are the packages that are installed:
# TBD: add multi-arch support for these binaries
#"packer=https://releases.hashicorp.com/packer/1.9.4/packer_1.9.4_linux_amd64.zip"
#"kubectl=https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.9/openshift-client-linux.tar.gz"
#"yq=https://github.com/mikefarah/yq/releases/download/v4.35.2/yq_linux_amd64.tar.gz"
#"umoci=https://github.com/opencontainers/umoci/releases/download/v0.4.7/umoci.amd64"

install_binary_packages() {
    # Define the required binary packages
    REQUIRED_BINARY_PACKAGES=(
        "packer=https://releases.hashicorp.com/packer/1.9.4/packer_1.9.4_linux_amd64.zip"
        "kubectl=https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.9/openshift-client-linux.tar.gz"
        "yq=https://github.com/mikefarah/yq/releases/download/v4.35.2/yq_linux_amd64.tar.gz"
        "umoci=https://github.com/opencontainers/umoci/releases/download/v0.4.7/umoci.amd64"
    )

    # Specify the installation directory
    local install_dir="/usr/local/bin"

    # Install the required binary packages
    for package_info in "${REQUIRED_BINARY_PACKAGES[@]}"; do
        IFS='=' read -r package_name package_url <<<"${package_info}"
        download_path="/tmp/${package_name}"

        if [[ -x "${install_dir}/${package_name}" ]]; then
            echo "Package ${package_name} is already installed. Skipping."
            continue
        else
            echo "Downloading ${package_name}..."
            curl -sSL "${package_url}" -o "${download_path}" ||
                error_exit "Failed to download ${package_name}"

            echo "Extracting ${package_name}..."
            if [[ "${package_url}" == *.zip ]]; then
                unzip -q "${download_path}" -d "${install_dir}" ||
                    error_exit "Failed to extract ${package_name}"
            elif [[ "${package_url}" == *.tar.gz ]]; then
                tar -xf "${download_path}" -C "${install_dir}" ||
                    error_exit "Failed to extract ${package_name}"
            else
                echo "Copying ${download_path} to ${install_dir}/${package_name}"
                cp "${download_path}" "${install_dir}/${package_name}" ||
                    error_exit "Failed to move ${package_name} to ${install_dir}"
            fi

            echo "Marking  ${install_dir}/${package_name} executable"
            # yq extracted file name is yq_linux_amd64. Rename it
            [[ "${package_name}" == "yq" ]] &&
                mv "${install_dir}/yq_linux_amd64" "${install_dir}/yq"

            chmod +x "${install_dir}/${package_name}" ||
                error_exit "Failed to mark ${package_name} executable"

            echo "Cleaning up..."
            rm -f "${download_path}"
        fi
    done

    echo "All binary packages installed successfully."

}

# Function to download source code from GitHub

function download_source_code() {

    # Download source code from GitHub
    # If any error occurs, exit the script with an error message

    # CAA_SRC_DIR is set to CAA_SRC_DOWNLOAD_DIR/src/cloud-api-adaptor
    # The default value of CAA_SRC_DOWNLOAD_DIR is /src/cloud-api-adaptor

    # Delete the source code download directory if it exists
    [[ -d "${CAA_SRC_DOWNLOAD_DIR}" ]] &&
        rm -rf "${CAA_SRC_DOWNLOAD_DIR}"

    # Create the root directory for source code
    mkdir -p "${CAA_SRC_DOWNLOAD_DIR}"

    # Download the source code from GitHub
    git clone "${CAA_SRC}" "${CAA_SRC_DOWNLOAD_DIR}" ||
        error_exit "Failed to download source code from GitHub"

    # Checkout the required commit
    cd "${CAA_SRC_DOWNLOAD_DIR}" ||
        error_exit "Failed to change directory to ${CAA_SRC_DOWNLOAD_DIR}"

    git checkout "${CAA_REF}" ||
        error_exit "Failed to checkout the required commit"

}

# Function to prepare the source code for building the image
# Patch any files that need to be patched
# Copy any files that need to be copied

function prepare_source_code() {

    # Prepare the source code for building the image
    # If any error occurs, exit the script with an error message

    # Ensure CAA_SRC_DIR is set
    [[ -z "${CAA_SRC_DIR}" ]] && error_exit "CAA_SRC_DIR is not set"

    local podvm_dir="${CAA_SRC_DIR}/podvm"

    mkdir -p "${podvm_dir}"/files

    # Download the podvm binaries and copy it to the podvm/files directory
    tar xvf /payload/podvm-binaries.tar.gz -C "${podvm_dir}"/files ||
        error_exit "Failed to download podvm binaries"

    # Set the NVIDIA_DRIVER_VERSION if variable is set
    if [[ "${NVIDIA_DRIVER_VERSION}" ]]; then
        echo "NVIDIA_DRIVER_VERSION is set to ${NVIDIA_DRIVER_VERSION}"
        sed -i "s/535/${NVIDIA_DRIVER_VERSION}/g" "${podvm_dir}"/addons/nvidia_gpu/setup.sh ||
            error_exit "Failed to set NVIDIA_DRIVER_VERSION"
    fi

    # Set the NVIDIA_USERSPACE_VERSION if variable is set
    if [[ "${NVIDIA_USERSPACE_VERSION}" ]]; then
        echo "NVIDIA_USERSPACE_VERSION is set to ${NVIDIA_USERSPACE_VERSION}"
        sed -i "s/1.13.5-1/${NVIDIA_USERSPACE_VERSION}/g" "${podvm_dir}"/addons/nvidia_gpu/setup.sh ||
            error_exit "Failed to set NVIDIA_USERSPACE_VERSION"
    fi

    if [[ "$BOOT_FIPS" == "yes" ]]; then
        echo "FIPS mode is enabled"
        sed -i '/exit 0/ifips-mode-setup --enable' "${podvm_dir}"/qcow2/misc-settings.sh ||
            error_exit "Failed to enable fips mode"
    fi

    if [[ "$CONFIDENTIAL_COMPUTE_ENABLED" == "yes" ]]; then
        sed 's/default SetPolicyRequest := true/default SetPolicyRequest := false/; s/default ExecProcessRequest := true/default ExecProcessRequest := false/' \
            "${podvm_dir}"/files/etc/kata-opa/default-policy.rego >"${podvm_dir}"/files/etc/kata-opa/coco-default-policy.rego
        ln -sf "${podvm_dir}"/files/etc/kata-opa/coco-default-policy.rego "${podvm_dir}"/files/etc/kata-opa/default-policy.rego

        # Remove the root disk mounts from the podvm
        rm -f "${podvm_dir}"/files/etc/systemd/system/{run-image,run-kata\\x2dcontainers}.mount
        rm -f "${podvm_dir}"/files/etc/systemd/system/multi-user.target.wants/{run-image,run-kata\\x2dcontainers}.mount
    fi
}

# Download and extract pause container image
# Accepts three arguments:
# 1. pause_image_repo_url: The registry URL of the OCP pause image.
# 2. pause_image_tag: The tag of the OCP pause image.
# 2. auth_json_file (optional): Path to the registry secret file to use for downloading the image
function download_and_extract_pause_image() {

    # Set default values for the OCP pause image
    pause_image_repo_url="${1:-${PAUSE_IMAGE_REPO_DEFAULT}}"
    pause_image_tag="${2:-${PAUSE_IMAGE_VERSION_DEFAULT}}"
    auth_json_file="${3:-${PAUSE_IMAGE_REPO_AUTH_FILE}}"

    # If arguments are not provided, exit the script with an error message
    [[ $# -lt 2 ]] &&
        error_exit "Usage: download_and_extract_pause_image <pause_image_repo_url> <pause_image_tag> [registry_secret]"

    # Ensure CAA_SRC_DIR is set
    [[ -z "${CAA_SRC_DIR}" ]] && error_exit "CAA_SRC_DIR is not set"

    local podvm_dir="${CAA_SRC_DIR}/podvm"
    local pause_src="/tmp/pause"
    local pause_bundle="${podvm_dir}/files/pause_bundle"

    mkdir -p "${pause_bundle}" ||
        error_exit "Failed to create the pause_bundle directory"

    # Form the skopeo CLI. Add authfile if provided
    if [[ -n "${3}" ]]; then
        SKOPEO_CLI="skopeo copy --authfile ${auth_json_file}"
    else
        SKOPEO_CLI="skopeo copy"
    fi

    # Download the pause image
    $SKOPEO_CLI "docker://${pause_image_repo_url}:${pause_image_tag}" "oci:${pause_src}:${pause_image_tag}" ||
        error_exit "Failed to download the pause image"

    # Extract the pause image using umoci into pause_bundle directory
    umoci unpack --rootless --image "${pause_src}:${pause_image_tag}" "${pause_bundle}" ||
        error_exit "Failed to extract the pause image"

}

# Download and extract container image
# Accepts three arguments:
# 1. image_repo_url: The registry URL of the container image
# 2. image_tag: The tag of the OCP pause image.
# 2. auth_json_file (optional): Path to the registry secret file to use for downloading the image
# Ensure you have sufficient disk space based on the image size
function download_and_extract_container_image() {

    local image_repo_url="${1}"
    local image_tag="${2}"
    local auth_json_file="${3}"

    # If arguments are not provided, return from the scrip by displaying a messaeg
    # No need to exit here as it's an optional step
    [[ $# -lt 2 ]] &&
        echo "Usage: download_and_extract_container_image <image_repo_url> <image_tag> [registry_secret]" &&
        return

    # Ensure CAA_SRC_DIR is set
    [[ -z "${CAA_SRC_DIR}" ]] && error_exit "CAA_SRC_DIR is not set"

    local podvm_dir="${CAA_SRC_DIR}/podvm"
    local image_src="tmp/image"

    local image_bundle="${podvm_dir}/files/image_bundle"

    mkdir -p "${image_bundle}" ||
        error_exit "Failed to create the image_bundle directory"

    # Form the skopeo CLI. Add authfile if provided
    # Need to use remove-signatures to avoid the error "Pushing signatures for OCI images is not supported"
    if [[ -n "${3}" ]]; then
        SKOPEO_CLI="skopeo copy --remove-signatures --authfile ${auth_json_file}"
    else
        SKOPEO_CLI="skopeo copy --remove-signatures"
    fi

    # Download the container image
    # This will copy the image to the image_src directory
    # ${image_src}/blobs/sha256 will contain the image layers
    # The layer names will be digest of the layer
    # eg
    # 3713021b02770a720dea9b54c03d0ed83e03a2ef5dce2898c56a327fee9a8bca
    # f634e1e7c42bc521c6a76ee114dadd709143cd4d66832806ac7b9503669d4752
    # These are tar archives of the layer contents
    $SKOPEO_CLI "docker://${image_repo_url}:${image_tag}" "oci:${image_src}:${image_tag}" ||
        error_exit "Failed to download the container image"

    # Get the sha256 digests of the image.
    # This will return a digest in the format "sha256:<sha256_digest>"
    # This will be an array
    IMG_DIGEST=$(skopeo inspect "docker://${image_repo_url}:${image_tag}" | jq '.Layers')

    # Exit if IMG_DIGEST is empty or null string
    [[ -z "${IMG_DIGEST}" || "${IMG_DIGEST}" = "null" ]] &&
        error_exit "Failed to get the sha256 digest of the image"

    # We need to copy the rootfs directory to the image_bundle directory in the following format
    # image_bundle/
    #   sha256_<sha256_digest>/{contents of the layer}

    # for each entry in the IMG_DIGEST array do the following
    # Remove quotes in the IMG_DIGEST and replace : with _
    # Create dir ${image_bundle}/${IMG_DIGEST}, extract and copy the layer contents to it
    # tar xvf ${image_src}/blobs/sha256/<sha256_digest> -C ${image_bundle}/${IMG_DIGEST}

    for i in $(echo "${IMG_DIGEST}" | jq -r '.[]'); do
        # The entry will be of the form sha256:<sha256_digest>

        # Create the image_bundle/sha256_<sha256_digest> directory
        layer_dir="${image_bundle}/${i/:/_}"
        mkdir -p "${layer_dir}" ||
            error_exit "Failed to create the ${layer_dir} directory"

        # Extract the sha256_digest
        i=$(echo "${i}" | cut -d ':' -f 2)

        # Extract the layer contents
        # We ignore the error here as we want to continue with the extraction
        # even if one of the layers fails to extract
        tar xvf "${image_src}/blobs/sha256/${i}" -C "${layer_dir}" ||
            echo "Failed to extract the layer contents for blob ${i}"
    done

    # Cleanup the tmp directories
    rm -rf "${image_src}"

    # Display the contents of the image_bundle directory
    ls -l "${image_bundle}"

    echo "Image extracted successfully and available in ${image_bundle}"

}

# Global variables

# Set global variable for the source code directory
# The project layout has changed for the cloud-api-adaptor project
# https://github.com/confidential-containers/cloud-api-adaptor
export CAA_SRC_DOWNLOAD_DIR="/src/cloud-api-adaptor"
export CAA_SRC_DIR="/src/cloud-api-adaptor/src/cloud-api-adaptor"
