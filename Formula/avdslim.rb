class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.2/avdslim_v1.0.2_darwin_arm64.tar.gz"
      sha256 "670600e10d5f36d44f3d91da25496b62c940d339de2e566b6e5bd6941ad4d5c5"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.2/avdslim_v1.0.2_darwin_amd64.tar.gz"
      sha256 "7e3e55c565cb37ac13c94170752b03e139c4d8ff588aff5228d236d900499de9"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.2/avdslim_v1.0.2_linux_arm64.tar.gz"
      sha256 "15d834e2fa5a2e8d5d816727609f6e2e28a65432ec708b73b69aec608b050b08"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.2/avdslim_v1.0.2_linux_amd64.tar.gz"
      sha256 "908c67c56f910587cd4e2af9fc5be3160c5571c6c6399c15b61db8149eaf700c"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
