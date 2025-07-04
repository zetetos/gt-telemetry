package main

import (
	"fmt"
	"log"
	"time"

	telemetry_client "github.com/zetetos/gt-telemetry"
	"github.com/zetetos/gt-telemetry/internal/telemetrysrc"
)

func main() {
	clientConfig := telemetry_client.GTClientOpts{
		Source:       "file://examples/simple/replay.gtz",
		Format:       telemetrysrc.TelemetryFormatTilde,
		StatsEnabled: true,
	}

	client, err := telemetry_client.NewGTClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create GT client: %s", err.Error())
	}

	go client.Run()

	fmt.Println("Waiting for data...    Press Ctrl+C to exit")

	sequenceID := uint32(0)
	for {
		if client.Finished {
			break
		}

		if sequenceID == client.Telemetry.SequenceID() {
			time.Sleep(8 * time.Millisecond)
			continue
		}
		sequenceID = client.Telemetry.SequenceID()

		suggestedGear := client.Telemetry.SuggestedGear()
		suggestedGearStr := fmt.Sprintf("[%d]", suggestedGear)
		if suggestedGear == 15 {
			suggestedGearStr = ""
		}

		hasTurbo := client.Telemetry.Flags().HasTurbo
		boostStr := ""
		if hasTurbo {
			boostStr = fmt.Sprintf("Boost: %+1.02f Bar", client.Telemetry.TurboBoostBar())
		}

		fmt.Print("\033[H\033[2J")
		fmt.Printf("Sequence ID:  %d\nTime of day:  %+v\n",
			client.Telemetry.SequenceID(),
			client.Telemetry.TimeOfDay(),
		)
		fmt.Printf("Race          Lap: %d of %d  Last lap: %+v  Best lap: %+v  Start position: %d  Race entrants: %d\n",
			client.Telemetry.CurrentLap(),
			client.Telemetry.RaceLaps(),
			client.Telemetry.LastLaptime(),
			client.Telemetry.BestLaptime(),
			client.Telemetry.StartingPosition(),
			client.Telemetry.RaceEntrants(),
		)

		fmt.Println()
		fmt.Printf("Vehicle       ID: %d  Name: %s %s  Drivetrain: %s  Aspiration: %s\n",
			client.Telemetry.VehicleID(),
			client.Telemetry.VehicleManufacturer(),
			client.Telemetry.VehicleModel(),
			client.Telemetry.VehicleDrivetrain(),
			client.Telemetry.VehicleAspirationExpanded(),
		)

		fmt.Println()
		fmt.Printf("Inputs        Throttle: %3.0f%% ➔ %3.0f%%  Brake: %3.0f%% ➔ %3.0f%%  Clutch: %3.0f%%  Steering: %+3.0f°  FFB: %+0.3f\n",
			client.Telemetry.ThrottlePedalPercent(),
			client.Telemetry.ThrottlePercent(),
			client.Telemetry.BrakePedalPercent(),
			client.Telemetry.BrakePercent(),
			client.Telemetry.ClutchActuationPercent(),
			client.Telemetry.SteeringWheelAngleDegrees(),
			client.Telemetry.SteeringWheelForceFeedback(),
		)

		fmt.Printf("Engine        Speed: %s rpm  %s  Energy recovery: %+3.03f\n",
			renderFlag(
				client.Telemetry.EngineRPMLight().Active,
				fmt.Sprintf("%0.0f", client.Telemetry.EngineRPM()),
				"yellow",
				"default",
			),
			boostStr,
			client.Telemetry.EnergyRecovery(),
		)
		fmt.Printf("Fluids        Fuel level: %3.0f%%  Fuel capacity: %3.0f%%  Water temp: %3.0f°c  Oil temp: %3.0f°c  Oil pressure: %3.02f kpa\n",
			client.Telemetry.FuelLevelPercent(),
			client.Telemetry.FuelCapacityPercent(),
			client.Telemetry.WaterTemperatureCelsius(),
			client.Telemetry.OilTemperatureCelsius(),
			client.Telemetry.OilPressureKPA(),
		)
		fmt.Printf("Clutch        Position: %3.0f%%  Engagement: %3.0f%%  Output: %5.0f RPM\n",
			client.Telemetry.ClutchActuationPercent(),
			client.Telemetry.ClutchEngagementPercent(),
			client.Telemetry.ClutchOutputRPM(),
		)
		fmt.Printf("Transmission  Gear: %2s %3s (%2d)          Ratios: 1[%0.03f]  3[%0.03f]  5[%0.03f]  7[%0.03f]\n",
			client.Telemetry.CurrentGearString(),
			suggestedGearStr,
			client.Telemetry.Transmission().Gears,
			client.Telemetry.Transmission().GearRatios[0],
			client.Telemetry.Transmission().GearRatios[2],
			client.Telemetry.Transmission().GearRatios[4],
			client.Telemetry.Transmission().GearRatios[6],
		)
		fmt.Printf("              vMax: %3d kph @ %5d rpm          2[%0.03f]  4[%0.03f]  6[%0.03f]  8[%0.03f] Diff[%0.03f] %f\n",
			client.Telemetry.CalculatedVmax().Speed,
			client.Telemetry.CalculatedVmax().RPM,
			client.Telemetry.Transmission().GearRatios[1],
			client.Telemetry.Transmission().GearRatios[3],
			client.Telemetry.Transmission().GearRatios[5],
			client.Telemetry.Transmission().GearRatios[7],
			client.Telemetry.DifferentialRatio(),
			client.Telemetry.TransmissionTopSpeedRatio(),
		)

		fmt.Println()
		fmt.Println("                    [  FL  ]  [  FR  ]  [  RL  ]  [  RR  ]")
		fmt.Printf("Suspension height:  [%5.0f ]  [%5.0f ]  [%5.0f ]  [%5.0f ] mm   Ride height: %0.02f mm\n",
			client.Telemetry.SuspensionHeightMillimeters().FrontLeft,
			client.Telemetry.SuspensionHeightMillimeters().FrontRight,
			client.Telemetry.SuspensionHeightMillimeters().RearLeft,
			client.Telemetry.SuspensionHeightMillimeters().RearRight,
			client.Telemetry.RideHeightMillimeters(),
		)
		fmt.Printf("Tyre temperature:   [%5.0f ]  [%5.0f ]  [%5.0f ]  [%5.0f ] °c\n",
			client.Telemetry.TyreTemperatureCelsius().FrontLeft,
			client.Telemetry.TyreTemperatureCelsius().FrontRight,
			client.Telemetry.TyreTemperatureCelsius().RearLeft,
			client.Telemetry.TyreTemperatureCelsius().RearRight,
		)
		fmt.Printf("Tyre diameter:      [%5.0f ]  [%5.0f ]  [%5.0f ]  [%5.0f ] mm\n",
			client.Telemetry.TyreDiameterMillimeters().FrontLeft,
			client.Telemetry.TyreDiameterMillimeters().FrontRight,
			client.Telemetry.TyreDiameterMillimeters().RearLeft,
			client.Telemetry.TyreDiameterMillimeters().RearRight,
		)
		fmt.Printf("Wheel RPM:          [%5.0f ]  [%5.0f ]  [%5.0f ]  [%5.0f ] rpm\n",
			client.Telemetry.WheelSpeedRPM().FrontLeft,
			client.Telemetry.WheelSpeedRPM().FrontRight,
			client.Telemetry.WheelSpeedRPM().RearLeft,
			client.Telemetry.WheelSpeedRPM().RearRight,
		)
		fmt.Printf("Wheel speed:        [%5.0f ]  [%5.0f ]  [%5.0f ]  [%5.0f ] kph  Ground speed: %0.0f kph\n",
			client.Telemetry.WheelSpeedKPH().FrontLeft,
			client.Telemetry.WheelSpeedKPH().FrontRight,
			client.Telemetry.WheelSpeedKPH().RearLeft,
			client.Telemetry.WheelSpeedKPH().RearRight,
			client.Telemetry.GroundSpeedKPH(),
		)
		fmt.Printf("Tyre slip ratio:    [%5s]  [%5s]  [%5s]  [%5s] %%\n",
			fmt.Sprintf("%+f", (client.Telemetry.TyreSlipRatio().FrontLeft-1)*100)[0:6],
			fmt.Sprintf("%+f", (client.Telemetry.TyreSlipRatio().FrontRight-1)*100)[0:6],
			fmt.Sprintf("%+f", (client.Telemetry.TyreSlipRatio().RearLeft-1)*100)[0:6],
			fmt.Sprintf("%+f", (client.Telemetry.TyreSlipRatio().RearRight-1)*100)[0:6],
		)
		fmt.Printf("Torque vector?:     [%+0.3f]  [%+0.3f]  [%+0.3f]  [%+0.3f]\n",
			client.Telemetry.Unknown0x140(),
			client.Telemetry.Unknown0x144(),
			client.Telemetry.Unknown0x148(),
			client.Telemetry.Unknown0x14C(),
		)

		fmt.Println()
		fmt.Println("                    [    X    ]  [    Y    ]  [    Z    ]")
		fmt.Printf("Position on map:    [%9s]  [%9s]  [%9s] m  Heading: %d\n",
			fmt.Sprintf("%+f", client.Telemetry.PositionalMapCoordinates().X)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.PositionalMapCoordinates().Y)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.PositionalMapCoordinates().Z)[0:9],
			int(client.Telemetry.Heading()*360),
		)
		fmt.Printf("Velocity:           [%9s]  [%9s]  [%9s] m/sec\n",
			fmt.Sprintf("%+f", client.Telemetry.VelocityVector().X)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.VelocityVector().Y)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.VelocityVector().Z)[0:9],
		)
		fmt.Printf("Angular velocity:   [%9s]  [%9s]  [%9s] rad/s\n",
			fmt.Sprintf("%+f", client.Telemetry.AngularVelocityVector().X)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.AngularVelocityVector().Y)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.AngularVelocityVector().Z)[0:9],
		)
		fmt.Printf("Rotation:           [%9s]  [%9s]  [%9s]\n",
			fmt.Sprintf("%+f", client.Telemetry.RotationVector().Pitch)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.RotationVector().Yaw)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.RotationVector().Roll)[0:9],
		)
		fmt.Printf("Translation:        [%9s]  [%9s]  [%9s]\n",
			fmt.Sprintf("%+f", client.Telemetry.TranslationEnvelope().Sway)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.TranslationEnvelope().Heave)[0:9],
			fmt.Sprintf("%+f", client.Telemetry.TranslationEnvelope().Surge)[0:9],
		)

		fmt.Println()
		fmt.Printf("Flags         %s    %s        %s\n",
			renderFlag(client.Telemetry.Flags().RevLimiterAlert, "RevLimit", "red", "grey"),
			renderFlag(client.Telemetry.Flags().TCSActive, "TCS", "red", "grey"),
			renderFlag(client.Telemetry.Flags().ASMActive, "ASM", "red", "grey"),
		)
		fmt.Printf("              %s      %s   %s\n",
			renderFlag(client.Telemetry.Flags().HeadlightsActive, "Lights", "green", "grey"),
			renderFlag(client.Telemetry.Flags().LowBeamActive, "Low beam", "yellow", "grey"),
			renderFlag(client.Telemetry.Flags().HighBeamActive, "High beam", "blue", "grey"),
		)
		fmt.Printf("              %s     %s\n",
			renderFlag(client.Telemetry.Flags().InGear, "In gear", "green", "red"),
			renderFlag(client.Telemetry.Flags().HandbrakeActive, "Handbrake", "red", "grey"),
		)
		fmt.Printf("              %s        %s    %s\n",
			renderFlag(client.Telemetry.Flags().Live, "Live", "green", "grey"),
			renderFlag(client.Telemetry.Flags().Loading, "Loading", "yellow", "grey"),
			renderFlag(client.Telemetry.Flags().GamePaused, "Paused", "red", "grey"),
		)
		fmt.Printf("Other flags   %s  %s  %s  %s\n",
			renderFlag(client.Telemetry.Flags().Flag13, "13", "red", "grey"),
			renderFlag(client.Telemetry.Flags().Flag14, "14", "red", "grey"),
			renderFlag(client.Telemetry.Flags().Flag15, "15", "red", "grey"),
			renderFlag(client.Telemetry.Flags().Flag16, "16", "red", "grey"),
		)

		fmt.Println()
		fmt.Printf("Unknowns      0x13E: %3d  0x13F: %3d  0x154: %+3.05f\n",
			client.Telemetry.Unknown0x13E(),
			client.Telemetry.Unknown0x13F(),
			client.Telemetry.Unknown0x154(),
		)

		if clientConfig.StatsEnabled {
			fmt.Println()
			fmt.Printf("Packet Size:  %d\n", client.Statistics.PacketSize)
			fmt.Printf("Packets       Total: %9d    Dropped: %9d     Invalid: %9d\n",
				client.Statistics.PacketsTotal,
				client.Statistics.PacketsDropped,
				client.Statistics.PacketsInvalid,
			)
			fmt.Printf("Packet rate   Current: %7d/s  Average: %9d/s   Maximum: %9d/s\n",
				client.Statistics.PacketRateCurrent,
				client.Statistics.PacketRateAvg,
				client.Statistics.PacketRateMax,
			)
			fmt.Printf("Decode time                       Average: %9dus   Maximum: %9dus\n",
				client.Statistics.DecodeTimeAvg.Microseconds(),
				client.Statistics.DecodeTimeMax.Microseconds(),
			)
		}

		timer := time.NewTimer(64 * time.Millisecond)
		<-timer.C
	}
}

func renderFlag(value bool, output string, trueColour string, fColour string) string {
	colour := fColour
	if value {
		colour = trueColour
	}

	switch colour {
	case "black":
		return fmt.Sprintf("\033[30m%s\033[0m", output)
	case "red":
		return fmt.Sprintf("\033[31m%s\033[0m", output)
	case "green":
		return fmt.Sprintf("\033[32m%s\033[0m", output)
	case "yellow":
		return fmt.Sprintf("\033[33m%s\033[0m", output)
	case "blue":
		return fmt.Sprintf("\033[34m%s\033[0m", output)
	case "magenta":
		return fmt.Sprintf("\033[35m%s\033[0m", output)
	case "cyan":
		return fmt.Sprintf("\033[36m%s\033[0m", output)
	case "white":
		return fmt.Sprintf("\033[37m%s\033[0m", output)
	case "grey":
		return fmt.Sprintf("\033[90m%s\033[0m", output)
	case "invisible":
		return ""
	default:
		return output
	}
}
