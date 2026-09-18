class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.11"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.11/avdslim_v1.0.11_darwin_arm64.tar.gz"
      sha256 "e304e6f3ff24a8f93f7e0a75edac14b33ae0870db79b5933ecdc24275b2b2c91"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.11/avdslim_v1.0.11_darwin_amd64.tar.gz"
      sha256 "eff036a466130d3aaba7a1d68122b8c9c8b1d31040fc8284638112389c9aad5a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.11/avdslim_v1.0.11_linux_arm64.tar.gz"
      sha256 "5d723f9f690ac74c3678113e040fecb28007f9864a16e4b52dcb81f929354b5e"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.11/avdslim_v1.0.11_linux_amd64.tar.gz"
      sha256 "9f696b99cc07a9b3935742ac429e37d898842d6da82ec9350e4ba63efbf636da"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
