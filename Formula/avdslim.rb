class Avdslim < Formula
  desc "Cut Android emulator (AVD) host RAM from ~8.5 GB to ~2.5 GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.14"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.14/avdslim_v1.0.14_darwin_arm64.tar.gz"
      sha256 "5889056bd9ee2653d12ba5b8a5464051191cf6740ce5e464fe1138901d969498"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.14/avdslim_v1.0.14_darwin_amd64.tar.gz"
      sha256 "9de331e2b0b44da7dc0957290847454a9f6e79ede33dc6f6ad626dd7d38ab910"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.14/avdslim_v1.0.14_linux_arm64.tar.gz"
      sha256 "3e7de720ca37ed7ca6828f245776870ceb5827b27969b556cc6b634487bd42ba"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.14/avdslim_v1.0.14_linux_amd64.tar.gz"
      sha256 "158ed1ad10ebde5802855e31392185b69ad89fecb760f1de9fb141956b00faa6"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
