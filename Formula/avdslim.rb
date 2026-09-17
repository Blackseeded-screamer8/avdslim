class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.9/avdslim_v1.0.9_darwin_arm64.tar.gz"
      sha256 "67094ba86373ee2d9d05b34b20a28ce14f6efb751c8551f7a8f3c91c6de24c04"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.9/avdslim_v1.0.9_darwin_amd64.tar.gz"
      sha256 "efeb20a1c625bac1d2cdf21e9fa5ea069ea890567b9da9344b34f43cc39a8c30"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.9/avdslim_v1.0.9_linux_arm64.tar.gz"
      sha256 "1cd21406b3802102cf8eb7045dccefa4428fd08f6a577300ee969a2dd9319694"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.9/avdslim_v1.0.9_linux_amd64.tar.gz"
      sha256 "3ad0983780b308c59ecb433702eccb9ca8b7529844b11c2ac71b7f7a8555217b"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
