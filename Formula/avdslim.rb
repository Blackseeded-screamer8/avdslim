class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.7"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.7/avdslim_v1.0.7_darwin_arm64.tar.gz"
      sha256 "e596deacc0f8c631fa05a34355be8f537cf66e30002e8f5d0d5a738aa3364f46"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.7/avdslim_v1.0.7_darwin_amd64.tar.gz"
      sha256 "fb9f54493b9f4d6a341a9bbe569d78dea574aa4d359fd5ebfb93f75037cacdc4"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.7/avdslim_v1.0.7_linux_arm64.tar.gz"
      sha256 "d84969dfb3c75e17f2754412d5cbd89a5b0f4d0d7852f41137c582f7868395cd"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.7/avdslim_v1.0.7_linux_amd64.tar.gz"
      sha256 "cc6d79d82f357fa2f19dfb198195db0f10a772eb9da8fbbe2e01cc55452d17ef"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
