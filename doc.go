// Package ofbx reads and writes FBX files.
//
// Load parses binary FBX 7.1 – 7.7 and text (ASCII) FBX into a Scene of
// typed objects (meshes with retained raw layers, materials, textures,
// skeletons, skins, blend shapes, animations, cameras, lights, embedded
// media). Malformed input returns errors instead of panicking.
//
// Three write paths cover the common workflows:
//
//   - Encode re-serializes a parsed element tree verbatim (binary), and
//     EncodeText does the same in the ASCII dialect.
//   - SaveScene exports a parsed scene after in-place edits applied via
//     the Set*/Sync*/Rebuild*/Rename helpers.
//   - A Document builds files from scratch (NewDocument, NewMesh,
//     NewMaterial, NewAnimationStack, ...) and writes them with Save.
//
// The package is dependency-light (testify in tests only) and its outputs
// are validated against third-party importers.
package ofbx
