# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Teash < Formula
  desc "A TUI browser for selecting and connecting to servers with Teleport"
  homepage "https://github.com/willgorman/teash"
  version "0.0.3"

  depends_on "teleport"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/willgorman/teash/releases/download/v0.0.3/teash_Darwin_x86_64.tar.gz", using: CurlDownloadStrategy
      sha256 "f38a7605667921d75e3037cc9bd2d262658c1c93e5d0ce2e2f44f8af199bbed3"

      def install
        bin.install "teash"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/willgorman/teash/releases/download/v0.0.3/teash_Darwin_arm64.tar.gz", using: CurlDownloadStrategy
      sha256 "d57f7ef4f733f4cac8db560b92ff1f96bd441479b9c1c4f8b221e2be6951ff4f"

      def install
        bin.install "teash"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/willgorman/teash/releases/download/v0.0.3/teash_Linux_arm64.tar.gz", using: CurlDownloadStrategy
      sha256 "181e27b22b1e4870beb8f506adb9145463b8e23d4542b410cf0c791db5b94e0d"

      def install
        bin.install "teash"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/willgorman/teash/releases/download/v0.0.3/teash_Linux_x86_64.tar.gz", using: CurlDownloadStrategy
      sha256 "7e009bcbaaa94689a0509c64c26d6ab1b9b24acab18188d7e576fcc9e8a3f0fd"

      def install
        bin.install "teash"
      end
    end
  end
end