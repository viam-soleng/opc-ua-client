# OPC UA Client modular resource

This module implements an OPC UA client which allows you to connect to an OPC UA server and read attributes.
The module is work in progress and additional features are planned to be implemented.
As always, please provide feedback and suggestions wherever applicable!

## Requirements

The module doesn't have any specific requirements. If you need support for another platform, let us know.

## Build and run

To use this module, follow the instructions to [add a module from the Viam Registry](https://docs.viam.com/registry/configure/#add-a-modular-resource-from-the-viam-registry) and select the `viam-soleng:opc-ua:opcsensor` model from the [opc-ua-client module](https://app.viam.com/module/viam-soleng/opc-ua-client).

## Configure your sensor

> [!NOTE]
> Before configuring your sensor you must [create a machine](https://docs.viam.com/manage/fleet/machines/#add-a-new-machine).

Navigate to the **Config** tab of your machine's page in [the Viam app](https://app.viam.com/).
Click on the **Components** subtab and click **Create component**.
Select the `<INSERT API NAME>` type, then select the `<INSERT MODEL>` model.
Click **Add module**, then enter a name for your sensor and click **Create**.

On the new component panel, copy and paste the following attribute template into your sensor’s **Attributes** box:

```json
{
  "nodeids": [
    "ns=2;i=3",
    "ns=2;i=2"
  ],
  "endpoint": "opc.tcp://0.0.0.0:4840/freeopcua/server/"
}
```

> [!NOTE]
> For more information, see [Configure a Machine](https://docs.viam.com/manage/configuration/).

### Attributes

The following attributes are available for `<INSERT API NAMESPACE>:<INSERT API NAME>:<INSERT MODEL>` sensor's:

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `todo1` | string | **Required** | TODO        |
| `todo2` | string | Optional     | TODO        |

### Example configuration

```json
{
  <INSERT SAMPLE CONFIGURATION(S)>
}
```

### Next steps

_Add any additional information you want readers to know and direct them towards what to do next with this module._
_For example:_

- To test your...
- To write code against your...

## Troubleshooting

_Add troubleshooting notes here._