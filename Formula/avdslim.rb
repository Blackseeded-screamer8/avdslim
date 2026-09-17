class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.10"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.10/avdslim_v1.0.10_darwin_arm64.tar.gz"
      sha256 "419e6f293c74b1d66a9b2d4e6a42ad5217287a35c1bb79db6690055dfc456e48"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.10/avdslim_v1.0.10_darwin_amd64.tar.gz"
      sha256 "c9ccfdfa7c569b3c7e5db4dcfd42391db22f6f79d5364abdca47b9ab27fd01d6"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.10/avdslim_v1.0.10_linux_arm64.tar.gz"
      sha256 "1ce229df75f35f3d10099522c2287cc4bae877446ff14a45e769c53b07936448"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.10/avdslim_v1.0.10_linux_amd64.tar.gz"
      sha256 "0b049e5ab365ac2074b59dec741713c5694bbcbedb854f3d521a2ba8c5fb3883"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
