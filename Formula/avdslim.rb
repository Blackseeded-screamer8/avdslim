class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.1/avdslim_v1.0.1_darwin_arm64.tar.gz"
      sha256 "76d62da5c54a8a9c8fcaec069b784e8e540a4ec332f579f6e1eeba11bee53c0f"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.1/avdslim_v1.0.1_darwin_amd64.tar.gz"
      sha256 "c06629c2a0b9546e6b6ffc3b03c514857acb2fcc6e480afe9b3e0644451c5517"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.1/avdslim_v1.0.1_linux_arm64.tar.gz"
      sha256 "d365c289adb8051296890a3ee2bcb078621e781065411e45b01feae4153ba2bb"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.1/avdslim_v1.0.1_linux_amd64.tar.gz"
      sha256 "aadce4142bcf056115bb629767ced5f78701db30efb428321db3d2444517987d"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
