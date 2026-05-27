IMAGE_NAME=gcr.io/lcpserver-1/github.com/readium/readium-lcp-server:latest
PORT=8080

if [ "$1" == 'update' ]; then
  ACTION=update-container
else
  echo "BE CAREFUL CREATE A DISK lcp-disk before" && exit 1
  ACTION=create-with-container
fi


echo "execute gcloud $ACTION"
gcloud compute instances $ACTION lcpserver-vm \
    --container-image $IMAGE_NAME \
    --container-env lSD_BASE_URL=http://127.0.0.1:8080/lsdserver,LCP_BASE_URL=http://127.0.0.1:8080/lcpserver,FRONTEND_BASE_URL=http://127.0.0.1:8080/frontend,BASE_URL=http://127.0.0.1:8080/frontend/,PORT=8080 \
    --disk name=lcp-disk \
    --container-mount-disk mount-path="/lcp",name=lcp-disk,mode=rw \
    --tags lcpserver

if [ "$1" != 'update' ]; then
  gcloud compute firewall-rules create allow-http \
    --allow tcp:$PORT --target-tags lcpserver
fi

