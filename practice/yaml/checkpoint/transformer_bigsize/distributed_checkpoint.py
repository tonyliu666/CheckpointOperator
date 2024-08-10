from ray import train
from ray.train import Checkpoint
from ray.train.torch import TorchTrainer


def train_func(config):
    ...

    with tempfile.TemporaryDirectory() as temp_checkpoint_dir:
        rank = train.get_context().get_world_rank()
        torch.save(
            ...,
            os.path.join(temp_checkpoint_dir, f"model-rank={rank}.pt"),
        )
        checkpoint = Checkpoint.from_directory(temp_checkpoint_dir)

        train.report(metrics, checkpoint=checkpoint)


trainer = TorchTrainer(
    train_func,
    scaling_config=train.ScalingConfig(num_workers=2),
    run_config=train.RunConfig(storage_path="s3://bucket/"),
)
# The checkpoint in cloud storage will contain: model-rank=0.pt, model-rank=1.pt