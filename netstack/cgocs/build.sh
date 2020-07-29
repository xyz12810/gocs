set -e
export CGO_ENABLED=1
export GOOS=darwin
export CGO_CFLAGS="-isysroot /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.5.sdk -miphoneos-version-min=7.0 -fembed-bitcode -arch arm64"
export CGO_CXXFLAGS="-isysroot /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.5.sdk -miphoneos-version-min=7.0 -fembed-bitcode -arch arm64"
export CGO_LDFLAGS="-isysroot /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.5.sdk -miphoneos-version-min=7.0 -fembed-bitcode -arch arm64"
export GOARCH=arm64
export CXX=/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang++
go build -tags ios -ldflags "-s -w" -buildmode=c-archive -o coversocks.a .
cp -f coversocks.a ~/deps/ios/arm64/lib/libcoversocks.a
cp -f coversocks.h ~/deps/ios/arm64/include/
