class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.0/avdslim_v1.0.0_darwin_arm64.tar.gz"
      sha256 "70833d73e3a7ea51d49e6f01476054fe07671b8cf3655544f064a848e8ee3cd1"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.0/avdslim_v1.0.0_darwin_amd64.tar.gz"
      sha256 "ef2b12232455530d371055ab05cb1f03e07e953edc2874be642f954a28edd915"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.0/avdslim_v1.0.0_linux_arm64.tar.gz"
      sha256 "35da1cac9319a1dd244d623daae4f77c4799745494fec38cf4fcdce538a462bd"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.0/avdslim_v1.0.0_linux_amd64.tar.gz"
      sha256 "e6bb765d99242397c0fa52d27757dd08a51e7311f737efdd10b5fb3506c17424"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
