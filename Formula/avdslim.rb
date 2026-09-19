class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.12"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.12/avdslim_v1.0.12_darwin_arm64.tar.gz"
      sha256 "0bf46f9e93c8373fd239f2990b311fe38cdbe0242932bba64c6dec0a84cdc3f1"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.12/avdslim_v1.0.12_darwin_amd64.tar.gz"
      sha256 "50a61ba4d5baff9ce35f46ef28e6f962ef24ac2df144c95db945d8b94304fd31"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.12/avdslim_v1.0.12_linux_arm64.tar.gz"
      sha256 "7aa82b0483d4bcca5fc6ab5155ba43f184611e9a825d5d81d304cc4128d78beb"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.12/avdslim_v1.0.12_linux_amd64.tar.gz"
      sha256 "a9a78120adab7f25009a3ffa18e755eacaff7f51205f014561c87e72ca5172bc"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
