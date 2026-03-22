BeginPackage["CoffeeLiqueur`Misc`Language`"]; 

LeakyModule::usage =
  "LeakyModule[vars, expr] works like Module[vars, expr], but leaks the fresh local \
symbols by appending them to a garbage collector list. The option \"Garbage\" :> g \
specifies where the held localized symbols are stored. This is useful for debugging, \
tracking generated symbols, or manually cleaning up temporary locals.";

AbortableTable::usage =
  "AbortableTable[expr, iter1, iter2, ...] generates a list of values from expr over \
the given iterators. If evaluation is aborted before completion, AbortableTable returns \
the partial results collected up to that point instead of discarding them.";


Begin["`Private`"]; 


Garbage = {}

ExtractFirst[x_, y___] := FakeHold[x];
SetAttributes[ExtractFirst, HoldFirst];
SetAttributes[FakeHold, HoldFirst];

LeakyModule[vars_, expr_, OptionsPattern[]] := With[{garbage = OptionValue[Automatic, Automatic, "Garbage", Unevaluated]},
    Module[vars, CompoundExpression[
        AppendTo[garbage, (Hold[vars] /. {Set :> ExtractFirst} // ReleaseHold) /. {FakeHold -> Hold}],
        expr
    ]]
]

SetAttributes[LeakyModule, HoldAll]
Options[LeakyModule] = {"Garbage" :> Garbage}

ClearAll[AbortableTable]

SetAttributes[AbortableTable, HoldAll];


AbortableTable[expr_, iter__] := Module[{tag = Unique["tag"], data = {}},
  data = Reap[
      CheckAbort[
        Do[
          Sow[expr, tag]
        , iter],
        Null
      ],
      tag
    ][[2]];
  If[data === {}, {}, First[data]]
]

End[];

EndPackage[];
