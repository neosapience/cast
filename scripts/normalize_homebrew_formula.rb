# frozen_string_literal: true

# GoReleaser always emits version; Homebrew infers it from our release URLs.
def normalize_formula(source)
  versions = source.scan(/^  version "([^"]+)"$/).flatten
  urls = source.scan(/^\s+url "([^"]+)"$/).flatten
  raise "Expected one version and four release URLs" unless versions.size == 1 && urls.size == 4
  unless urls.all? { |url| url.start_with?("https://github.com/neosapience/cast/releases/download/v#{versions.first}/") }
    raise "Release URLs must match the explicit version"
  end

  source.sub(/^  version "[^"]+"\n/, "")
end

if ARGV == ["--self-test"]
  fixture = "  version \"1.0.9\"\n" + (["  url \"https://github.com/neosapience/cast/releases/download/v1.0.9/cast.tar.gz\"\n"] * 4).join
  raise "Version not removed" if normalize_formula(fixture).include?("  version ")
  raise "Other content changed" unless normalize_formula(fixture) == fixture.lines.drop(1).join
  begin
    normalize_formula(fixture.sub("v1.0.9/", "v1.0.8/"))
    raise "Accepted mismatched release URL"
  rescue RuntimeError => error
    raise unless error.message == "Release URLs must match the explicit version"
  end
  puts "Homebrew formula checks passed"
else
  abort "Usage: ruby #{$PROGRAM_NAME} FORMULA | --self-test" unless ARGV.size == 1
  path = ARGV.first
  File.write(path, normalize_formula(File.read(path)))
end
