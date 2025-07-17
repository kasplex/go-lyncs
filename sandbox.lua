(function()
--[[-code-callbacks-]]
	local _set = function (t)
		local _setmt=setmetatable; if t==_G then setmetatable=nil; getmetatable=nil end
		local mt = {
			__index=t, __newindex=function(_, k, v)
				if t~=_G then error("variable read-only",2) end
				if fn[k] and t[k]==nil and type(v)=="function" then t[k]=v; return end
				error("variable read-only", 2)
			end
		}
		return _setmt({}, mt)
	end
	table = _set(table)
	string = _set(string)
	math = _set(math)
	bit = _set(bit)
	mpz = _set(mpz)
--[[-code-readonly-list-]]
--[[-code-debug-]]
	setfenv(2,_set(_G)); setfenv=nil
end)(); local _;
