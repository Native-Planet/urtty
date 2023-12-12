#!/bin/bash
cd urtty-fe
DOCKER_BUILDKIT=0 docker build -t urtty-builder .
cd ..
container_id=$(docker create urtty-builder)
rm -rf web
docker cp $container_id:/webui/build ./web
curl https://bootstrap.urbit.org/globberv3.tgz | tar xzk
./zod/.run -d
dojo () {
  curl -s --data '{"source":{"dojo":"'"$1"'"},"sink":{"stdout":null}}' http://localhost:12321    
}
hood () {
  curl -s --data '{"source":{"dojo":"+hood/'"$1"'"},"sink":{"app":"hood"}}' http://localhost:12321    
}
mkdir -p zod/work/glob
cp -r web/* zod/work/glob
hood "commit %work"
dojo "-garden!make-glob %work /glob"
hash=$(ls -1 -c zod/.urb/put | head -1 | sed "s/glob-\([a-z0-9\.]*\).glob/\1/")
sed -i "s/glob\-[a-z0-9\.]*glob' *[a-z0-9\.]*\]/glob-$hash.glob' $hash]/g" $2
cp zod/.urb/put/*.glob .
hood "exit"
sleep 5s
rm -rf zod