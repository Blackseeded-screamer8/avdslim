class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.3/avdslim_v1.0.3_darwin_arm64.tar.gz"
      sha256 "8289ed854b3b88f548a90320096103ebd8a9bc18bd7d2e223ca48dc3ff936df8"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.3/avdslim_v1.0.3_darwin_amd64.tar.gz"
      sha256 "046056ae8cf66d626dfc8147b77512fbcff4952a9e73d3896281e642607be965"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.3/avdslim_v1.0.3_linux_arm64.tar.gz"
      sha256 "1880c70c5377281b79bfc4700b2ae33cdb656cd39ec061303e9d3acb43f60d64"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.3/avdslim_v1.0.3_linux_amd64.tar.gz"
      sha256 "2d07bbe6b15ade5be2e2f9ae84ff146039421882e7c43d5f9c88594f88280d8c"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
