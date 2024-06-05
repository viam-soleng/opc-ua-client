import asyncio

from viam.robot.client import RobotClient
from viam.rpc.dial import DialOptions
from viam.components.sensor import Sensor

async def connect():
    options = RobotClient.Options(dial_options=DialOptions(insecure=True,disable_webrtc=True))
    return await RobotClient.at_address('0.0.0.0:8083', options)

async def main():
    machine = await connect()

    #print('Resources:')
    #print(machine.resource_names)
    
    # opc-sensor
    opc_sensor = Sensor.from_robot(machine, "opc-ua-sensor")
    opc_sensor_return_value = await opc_sensor.get_readings()
    print(f"opc-sensor get_readings return value: {opc_sensor_return_value}")

    result = await opc_sensor.do_command({"write":{'ns=2;i=3':11,'ns=2;i=2':11}})
    print(f"Result: {result}")
    opc_sensor_return_value = await opc_sensor.get_readings()
    print(f"opc-sensor get_readings return value: {opc_sensor_return_value}")

    # Don't forget to close the machine when you're done!
    await machine.close()

if __name__ == '__main__':
    asyncio.run(main())
