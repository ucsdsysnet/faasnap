
import datetime, json, os, uuid
from time import time
from PIL import Image
import torch
from torchvision import transforms
from torchvision.models import resnet50

tmp = '/dev/shm/'

SCRIPT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__)))
class_idx = json.load(open(os.path.join(SCRIPT_DIR, "imagenet_class_index.json"), 'r'))
idx2label = [class_idx[str(k)][1] for k in range(len(class_idx))]
model = None

def lambda_handler(event, ctx):
    r = ctx['r']
  
    input_object_key = event.get('input_object_key')
    output_object_key = event.get('output_object_key')
    model_key = event.get('model_object_key')
    os.makedirs(tmp, exist_ok=True)
    image_path = '%s/%s' % (tmp, input_object_key)
    with open(image_path, 'wb') as f:
        f.write(r.get(input_object_key))

    global model
    if not model:
        model_path = os.path.join(tmp, model_key)
        with open('%s/%s' % (tmp, model_key), 'wb') as f:
            f.write(r.get(model_key))

        model = resnet50(pretrained=False)
        model.load_state_dict(torch.load(model_path))
        model.eval()


    ts1 = time()
    input_image = Image.open(image_path)
    preprocess = transforms.Compose([
        transforms.Resize(256),
        transforms.CenterCrop(224),
        transforms.ToTensor(),
        transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
    ])
    input_tensor = preprocess(input_image)
    input_batch = input_tensor.unsqueeze(0) # create a mini-batch as expected by the model 
    output = model(input_batch)
    _, index = torch.max(output, 1)
    # The output has unnormalized scores. To get probabilities, you can run a softmax on it.
    prob = torch.nn.functional.softmax(output[0], dim=0)
    _, indices = torch.sort(output, descending = True)
    ret = idx2label[index]
    ts2 = time()
    
    return [ts1, ts2]
