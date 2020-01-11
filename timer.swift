#!/usr/bin/swift
import Swift
import Cocoa

let minutes = CommandLine.arguments[1]

print("Starting " + minutes + " minutes timer")

let window = NSWindow(contentRect: NSMakeRect(100, 100, 600, 200),
styleMask:NSWindow.StyleMask.borderless,
                           backing: NSWindow.BackingStoreType.buffered, defer: true)
window.level = .floating
let controller = NSWindowController(window: window)
controller.showWindow(window)

print("Finished " + minutes + " minutes timer")
