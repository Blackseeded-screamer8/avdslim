class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.8"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.8/avdslim_v1.0.8_darwin_arm64.tar.gz"
      sha256 "f36ab9aa32f8bc538942f9e09e9f0b559468a667bde075adc537c1bc9d9a7f06"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.8/avdslim_v1.0.8_darwin_amd64.tar.gz"
      sha256 "9438911be1c647d6413bae1c00876f07e9f7f39936f86c06d01d19744ff4f415"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.8/avdslim_v1.0.8_linux_arm64.tar.gz"
      sha256 "dfccb91a5cf9ae1b4c143768d31e6d03eece3dcb540ef20ade55ce9100e43f05"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.8/avdslim_v1.0.8_linux_amd64.tar.gz"
      sha256 "a445d243abdd3007327867581f35b8641614de3c1f8cd904d5b52eb539898876"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
