# Ruby script that builds binaries for 64bit windows, osx and linux
builds = [
	["darwin", "amd64", "zip", "_osx"],
	["linux", "amd64", "tar.gz", "_linux_amd64"],
	["windows", "amd64", "zip", "_win"]
]
Dir.mkdir("release") unless Dir.exist?("release")
builds.each {|info|
	os = info[0]
	arch = info[1]
	buildCmd = "GOOS=#{os} GOARCH=#{arch} go build"
	ext = (os == "windows")? ".exe" : ""
	tag = info[3]
	deployCmd = ""
	if (info[2] == "zip") 
		zipFile = "elktail#{tag}.zip"
		deployCmd = "zip release/#{zipFile} elktail#{ext}"
	else
		zipFile = "elktail#{tag}.tar.gz"
		deployCmd = "tar cvzf release/#{zipFile} elktail#{ext}"
	end
	puts "Building: " + buildCmd
	puts system(buildCmd)
	puts "Deploying: " + deployCmd
	puts system(deployCmd)
}
