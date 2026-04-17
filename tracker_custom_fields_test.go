package mtag

import "testing"

func TestCapabilities_TrackerCustomFieldsUnsupported(t *testing.T) {
	for _, kind := range []ContainerKind{ContainerMOD, ContainerS3M, ContainerXM, ContainerIT} {
		t.Run(kind.String(), func(t *testing.T) {
			f := testFileForKind(kind)
			caps := f.Capabilities()
			if caps.CustomFields.Read || caps.CustomFields.Write {
				t.Fatalf("tracker custom fields = %+v, want unsupported", caps.CustomFields)
			}
		})
	}
}
