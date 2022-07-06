IMAGE_NAME=hello
PORT=8080

if [ $1 == 'update' ]; then
  ACTION=update-container
else
  ACTION=create-with-container
fi

gcloud compute instances $ACTION lcpserver-vm \
    --container-image $IMAGE_NAME \
    --container-env lSD_BASE_URL=http://127.0.0.1:8080/lsdserver,LCP_BASE_URL=http://127.0.0.1:8080/lcpserver,FRONTEND_BASE_URL=http://127.0.0.1:8080/frontend,BASE_URL=http://127.0.0.1:8080/frontend/ \
    --disk name=lcp-disk \
    --container-mount-disk mount-path="/lcp",name=lcp-disk,mode=rw \
    --tags lcpserver

gcloud compute firewall-rules create allow-http \
   --allow tcp:$PORT --target-tags lcpserver

