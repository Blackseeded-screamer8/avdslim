class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  url "https://github.com/kdbhalala/avdslim/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/kdbhalala/avdslim.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
